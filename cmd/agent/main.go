package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	agent "github.com/lisanmuaddib/agent-go/pkg"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/llm/openai"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenAI client
	openaiConfig, err := openai.NewOpenAIConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI config")
	}

	llmClient, err := openai.NewOpenAIClient(openaiConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create OpenAI client")
	}

	// Initialize Twitter config using the provided function
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

	// Initialize agent
	agent, err := agent.New(agent.Config{
		LLM:           llmClient.GetLLM(),
		TwitterClient: twitterClient,
		Logger:        log,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create agent")
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()
	}()

	log.Info("Starting Twitter mention monitoring")

	// Run the agent
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		log.WithError(err).Fatal("Agent stopped with error")
	}

	log.Info("Agent shutdown complete")
}
