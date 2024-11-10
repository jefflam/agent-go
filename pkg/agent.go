package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/lisanmuaddib/agent-go/pkg/actions"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

type Agent struct {
	client  *twitter.TwitterClient
	llm     llms.Model
	logger  *logrus.Logger
	actions map[string]actions.Action
	mu      sync.RWMutex
}

type Config struct {
	LLM           llms.Model
	Logger        *logrus.Logger
	TwitterClient *twitter.TwitterClient
	TweetStore    *memory.TweetStore
}

func New(config Config) (*Agent, error) {
	if config.LLM == nil {
		return nil, fmt.Errorf("LLM is required")
	}
	if config.TwitterClient == nil {
		return nil, fmt.Errorf("TwitterClient is required")
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	return &Agent{
		client:  config.TwitterClient,
		llm:     config.LLM,
		logger:  config.Logger,
		actions: make(map[string]actions.Action),
	}, nil
}

// RegisterAction adds a new action to the agent
func (a *Agent) RegisterAction(action actions.Action) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := action.Name()
	if _, exists := a.actions[name]; exists {
		return fmt.Errorf("action %s already registered", name)
	}

	a.actions[name] = action
	return nil
}

// Run starts all registered actions
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("Starting agent with registered actions")

	// Create error channel for collecting errors from actions
	errChan := make(chan error, len(a.actions))

	// Start each action in its own goroutine
	var wg sync.WaitGroup
	for name, action := range a.actions {
		wg.Add(1)
		go func(name string, action actions.Action) {
			defer wg.Done()

			a.logger.WithField("action", name).Info("Starting action")
			if err := action.Execute(ctx); err != nil {
				a.logger.WithError(err).WithField("action", name).Error("Action failed")
				errChan <- fmt.Errorf("action %s failed: %w", name, err)
			}
		}(name, action)
	}

	// Wait for context cancellation or errors
	select {
	case <-ctx.Done():
		a.logger.Info("Context cancelled, stopping all actions")
		a.stopAllActions()
		return ctx.Err()
	case err := <-errChan:
		a.logger.WithError(err).Error("Action error occurred")
		a.stopAllActions()
		return err
	}
}

// stopAllActions cleanly stops all registered actions
func (a *Agent) stopAllActions() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for name, action := range a.actions {
		a.logger.WithField("action", name).Info("Stopping action")
		action.Stop()
	}
}
