package agent

import (
	"context"
	"sync"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// TaskType represents different types of agent tasks
type TaskType string

// Let's also update the TaskConfig struct to include metadata
type TaskConfig struct {
	Enabled  bool
	Interval time.Duration
	Metadata TaskMetadata
}

// Task represents a runnable agent task
type Task interface {
	Run(ctx context.Context) error
	Stop()
	Type() TaskType
}

// Agent represents a Twitter AI agent that manages multiple tasks
type Agent struct {
	client      *twitter.TwitterClient
	llm         llms.Model
	logger      *logrus.Logger
	tasks       map[TaskType]Task
	tasksMu     sync.RWMutex
	taskConfigs map[TaskType]TaskConfig
}

// Config holds the configuration for the Agent
type Config struct {
	LLM           llms.Model
	Logger        *logrus.Logger
	TwitterClient *twitter.TwitterClient
	Tasks         map[TaskType]TaskConfig
}

// AgentOption defines functional options for configuring the agent
type AgentOption func(*Agent) error

// NewAgent creates a new agent with the given options
func NewAgent(config Config, opts ...AgentOption) (*Agent, error) {
	agent := &Agent{
		client:      config.TwitterClient,
		llm:         config.LLM,
		logger:      config.Logger,
		tasks:       make(map[TaskType]Task),
		taskConfigs: config.Tasks,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, err
		}
	}

	// Validate task configurations
	if err := ValidateTaskConfigs(agent.taskConfigs); err != nil {
		return nil, err
	}

	return agent, nil
}
