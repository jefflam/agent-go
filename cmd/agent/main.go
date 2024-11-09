package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	// Initialize Twitter config using the provided function
	log.Info("Initializing Twitter client")
	twitterConfig, err := twitter.NewTwitterConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to create Twitter config")
	}
	// Override logger to use our main logger
	twitterConfig.Logger = log

	// Initialize Twitter client with config
	twitterClient, err := twitter.NewTwitterClient(twitterConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Twitter client")
	}

	// Initialize TweetStore with database
	log.Info("Initializing TweetStore")
	tweetStore, err := memory.NewTweetStore(log, database)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize tweet store")
	}

	// Initialize agent
	log.Info("Initializing agent")
	agent, err := agent.New(agent.Config{
		LLM:           llmClient.GetLLM(),
		TwitterClient: twitterClient,
		Logger:        log,
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
