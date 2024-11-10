package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/internal/agentconfig"
	agent "github.com/lisanmuaddib/agent-go/pkg"
	"github.com/lisanmuaddib/agent-go/pkg/db"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/llm/openai"
	"github.com/lisanmuaddib/agent-go/pkg/logging"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/sirupsen/logrus"
)

// Initialize Twitter client and get bot ID with rate limit handling
func initializeTwitterClient(ctx context.Context, log *logrus.Logger) (*twitter.TwitterClient, string, error) {
	log.Info("Initializing Twitter client")
	twitterConfig, err := twitter.NewTwitterConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Twitter config: %w", err)
	}
	twitterConfig.Logger = log

	// Check if UserID is configured in env
	if twitterConfig.UserID != "" {
		log.WithFields(logrus.Fields{
			"user_id": twitterConfig.UserID,
			"source":  "env",
		}).Info("Using configured Twitter user ID from environment")
		twitterClient, err := twitter.NewTwitterClient(twitterConfig)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Twitter client: %w", err)
		}
		return twitterClient, twitterConfig.UserID, nil
	}

	// If no UserID configured, proceed with API call and rate limit handling
	twitterClient, err := twitter.NewTwitterClient(twitterConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Twitter client: %w", err)
	}

	// Get bot ID with rate limit handling
	botID, err := twitterClient.GetAuthenticatedUserID(ctx)
	if err != nil {
		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "rate limit exceeded") {
			log.WithError(err).Warning("Rate limit hit during initialization - bot ID will be fetched later")
			// Return client without botID - it will be fetched later when rate limit resets
			return twitterClient, "", nil
		}
		return nil, "", fmt.Errorf("failed to get bot user ID: %w", err)
	}

	log.WithFields(logrus.Fields{
		"bot_id": botID,
		"source": "api",
	}).Info("Successfully retrieved bot ID during initialization")

	return twitterClient, botID, nil
}

// Helper function to extract reset time from error message
func extractResetTime(errMsg string) string {
	// Example error message format: "rate limit exceeded, reset in 23h59m17s at 2024-11-11T00:31:43Z"
	parts := strings.Split(errMsg, " at ")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// Add this simple env config implementation
type envConfig struct{}

func (e *envConfig) GetString(key string) string {
	return os.Getenv(key)
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Only log warning since .env is optional
		logrus.WithError(err).Warn("Error loading .env file")
	}

	// Initialize logger with colored formatter
	log := logrus.New()
	log.SetFormatter(logging.NewColoredJSONFormatter())

	// Get log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if level, err := logrus.ParseLevel(logLevel); err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(logrus.InfoLevel)
		log.WithFields(logrus.Fields{
			"attempted_level": logLevel,
			"default_level":   "INFO",
		}).Warn("Invalid log level specified, defaulting to INFO")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	log.Info("Initializing database connection")
	database, err := db.SetupDatabase(log)
	if err != nil {
		log.WithError(err).Fatal("Failed to setup database connection")
	}

	// Get underlying *sql.DB to ensure clean shutdown
	sqlDB, err := database.DB()
	if err != nil {
		log.WithError(err).Fatal("Failed to get underlying database connection")
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.WithError(err).Error("Error closing database connection")
		}
		log.Info("Database connection closed")
	}()

	// Initialize OpenAI client
	log.Info("Initializing OpenAI client")
	openaiConfig, err := openai.NewOpenAIConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI config")
	}

	llmClient, err := openai.NewOpenAIClient(openaiConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI client")
	}

	// Initialize Twitter client with rate limit handling
	twitterClient, botID, err := initializeTwitterClient(ctx, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Twitter client")
	}

	// Create simple env config
	env := &envConfig{}

	// Initialize TweetStore with botID (if available)
	var tweetStore *memory.TweetStore
	if botID != "" {
		log.WithField("bot_id", botID).Info("Initializing TweetStore with bot ID")
		tweetStore, err = memory.NewTweetStore(log, database, botID, env)
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize tweet store")
		}
	} else {
		log.Info("TweetStore initialization delayed until bot ID is available")
		// Initialize with a placeholder - will be updated when we get the bot ID
		tweetStore, err = memory.NewTweetStore(log, database, "pending", env)
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize tweet store")
		}
	}

	// Initialize agent
	log.Info("Initializing agent")
	agent, err := agent.New(agent.Config{
		LLM:           llmClient.GetLLM(),
		TwitterClient: twitterClient,
		Logger:        log,
		TweetStore:    tweetStore,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create agent")
	}

	// Configure and register actions
	log.Info("Configuring agent actions")
	actions, err := agentconfig.ConfigureActions(agentconfig.ActionConfig{
		TwitterClient: twitterClient,
		LLM:           llmClient.GetLLM(),
		Logger:        log,
		TweetStore:    tweetStore,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to configure actions")
	}

	for _, action := range actions {
		if err := agent.RegisterAction(action); err != nil {
			log.WithError(err).Fatal("Failed to register action")
		}
	}

	// If we don't have botID yet, start a goroutine to fetch it when rate limit resets
	if botID == "" {
		go func() {
			for {
				id, err := twitterClient.GetAuthenticatedUserID(ctx)
				if err != nil {
					if strings.Contains(err.Error(), "rate limit exceeded") {
						log.WithError(err).Warning("Still rate limited, will retry later")
						// Parse the reset time from the error message
						if resetStr := extractResetTime(err.Error()); resetStr != "" {
							resetTime, parseErr := time.Parse(time.RFC3339, resetStr)
							if parseErr == nil {
								waitDuration := time.Until(resetTime)
								log.WithField("wait_duration", waitDuration.Round(time.Second)).
									Info("Waiting for rate limit reset")
								time.Sleep(waitDuration)
								continue
							}
						}
						// If we couldn't parse the reset time, use a default backoff
						time.Sleep(5 * time.Minute)
						continue
					}
					log.WithError(err).Error("Failed to get bot ID, will retry in 5 minutes")
					time.Sleep(5 * time.Minute)
					continue
				}

				// Successfully got the bot ID
				log.WithFields(logrus.Fields{
					"bot_id": id,
					"source": "api_retry",
				}).Info("Successfully retrieved bot ID after rate limit reset")
				if err := tweetStore.UpdateBotID(ctx, id); err != nil {
					log.WithError(err).Error("Failed to update tweet store with bot ID")
				}
				return
			}
		}()
	}

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("Received shutdown signal")

		// Begin graceful shutdown
		log.Info("Starting graceful shutdown")

		// Close database connection gracefully
		if err := sqlDB.Close(); err != nil {
			log.WithError(err).Error("Error closing database connection during shutdown")
		}

		cancel() // Cancel context to stop all operations
	}()

	log.Info("Starting Twitter mention monitoring")

	// Run the agent
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		log.WithError(err).Fatal("Agent stopped with error")
	}

	log.Info("Agent shutdown complete")
}
