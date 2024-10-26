#!/bin/bash

# Create main directory structure
mkdir -p cmd/agent
mkdir -p internal/{personality/{traits,behaviors},store}
mkdir -p pkg/{interfaces/twitter,thoughts/{context,memory,reasoning,emotions},actions,llm/openai,prompts/templates}
mkdir -p tests/{integration,e2e}

# Initialize go.mod
go mod init github.com/brendanplayford/agent-go

# Create initial files with basic content and comments
touch cmd/agent/main.go                     # Application entry point

# Internal package files
touch internal/personality/traits/traits.go          # Agent personality traits
touch internal/personality/behaviors/behaviors.go     # Agent behaviors
touch internal/store/store.go                        # Data persistence

# Pkg package files
touch pkg/interfaces/twitter/config.go               # Twitter-specific configs
touch pkg/thoughts/config.go                         # Thought processing configs
touch pkg/thoughts/processor.go                      # Main thought processing engine
touch pkg/thoughts/context/context.go                # Context awareness
touch pkg/thoughts/memory/memory.go                  # Short/long term memory
touch pkg/thoughts/reasoning/reasoning.go            # Decision making logic
touch pkg/thoughts/emotions/emotions.go              # Emotional state handling
touch pkg/actions/config.go                          # Action configurations
touch pkg/llm/config.go                             # LLM provider configs
touch pkg/llm/openai/config.go                      # OpenAI-specific configs
touch pkg/prompts/config.go                         # Prompt configurations
touch pkg/prompts/templates/base.go                 # Reusable prompt templates

# Test directories
touch tests/integration/.gitkeep
touch tests/e2e/.gitkeep

# Create main.go with basic content
cat > cmd/agent/main.go << 'EOF'
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
        logLevel = "INFO" // Default to INFO level
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
EOF

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
vendor/

# Go workspace file
go.work

# Environment files
.env
.env.local

# IDE specific files
.idea/
.vscode/
*.swp
*.swo

# OS specific files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
EOF

# Initialize go.mod dependencies
go mod tidy
go get github.com/sirupsen/logrus
go get github.com/tmc/langchaingo
