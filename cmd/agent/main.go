package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lisanmuaddib/agent-go/pkg/llm/openai"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

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

	// Initialize OpenAI client
	config, err := openai.NewOpenAIConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI config")
	}
	config.Logger = log

	client, err := openai.NewOpenAIClient(config)
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI client")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.WithFields(logrus.Fields{
			"signal": sig.String(),
		}).Info("Received shutdown signal")
		cancel()
	}()

	log.WithFields(logrus.Fields{
		"service": "twitter-agent",
		"version": "0.1.0",
	}).Info("Starting Twitter Agent")

	// Example LLM interaction
	prompt := "What's an interesting topic to tweet about regarding artificial intelligence?"
	response, err := client.Generate(ctx, prompt)
	if err != nil {
		log.WithError(err).Error("Failed to generate response")
	} else {
		log.WithFields(logrus.Fields{
			"prompt":   prompt,
			"response": response,
		}).Info("Generated response successfully")
	}

	<-ctx.Done()
	log.Info("Shutting down gracefully...")
}
