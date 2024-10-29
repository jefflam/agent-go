package actions

import (
	"context"
	"time"
)

// Action represents a single action that can be performed by the agent
type Action interface {
	// Name returns the unique identifier for this action
	Name() string
	// Execute runs the action with the given context
	Execute(ctx context.Context) error
	// Stop cleanly stops the action
	Stop()
}

// ActionConfig holds common configuration for actions
type ActionConfig struct {
	Name     string
	Interval time.Duration
}
