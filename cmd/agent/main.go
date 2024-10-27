package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

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

	// TODO: Initialize and start agent components
	<-ctx.Done()
	log.Info("Shutting down gracefully...")
}
