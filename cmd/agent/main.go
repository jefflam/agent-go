package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/brendanplayford/agent-go/pkg/thoughts"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

// ThoughtProcessor handles the creation and processing of thoughts
type ThoughtProcessor interface {
	Process(ctx context.Context, input string) (*thoughts.Thought, error)
	Shutdown() error
}

// Config holds the application configuration
type Config struct {
	LogLevel string
	// Add other config options as needed
}

func init() {
	// Initialize logger
	log.SetFormatter(&logrus.JSONFormatter{})

	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(level)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize thought processor
	processor, err := thoughts.NewProcessor(thoughts.Config{
		Logger: log,
		// Add other configuration options
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize thought processor")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.WithFields(logrus.Fields{
			"signal": sig.String(),
		}).Info("Received shutdown signal")
		if err := processor.Shutdown(); err != nil {
			log.WithError(err).Error("Error during thought processor shutdown")
		}
		cancel()
	}()

	log.WithFields(logrus.Fields{
		"service": "twitter-agent",
		"version": "0.1.0",
	}).Info("Starting Twitter Agent")

	// Example thought processing
	thought, err := processor.Process(ctx, "What should I tweet about today?")
	if err != nil {
		log.WithError(err).Error("Failed to process thought")
	} else {
		log.WithFields(logrus.Fields{
			"thoughtID": thought.ID,
			"content":   thought.Content,
		}).Info("Thought processed successfully")
	}

	<-ctx.Done()
	log.Info("Shutting down gracefully...")
}
