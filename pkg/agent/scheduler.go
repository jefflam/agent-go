package agent

import (
	"fmt"
	"time"
)

const (
	// DefaultMentionCheckInterval is the default duration between mention checks
	DefaultMentionCheckInterval = 30 * time.Second

	// MinMentionCheckInterval is the minimum allowed interval to prevent rate limiting
	MinMentionCheckInterval = 30 * time.Second
	// MaxMentionCheckInterval is the maximum allowed interval to ensure responsiveness
	MaxMentionCheckInterval = 5 * time.Minute
)

// TaskConfig holds configuration for a scheduled task
type TaskConfig struct {
	Interval time.Duration
	Name     string
	Enabled  bool
}

// SchedulerConfig holds timing configurations for various agent tasks
type SchedulerConfig struct {
	MentionCheckInterval time.Duration
	Tasks                map[string]TaskConfig
}

// NewDefaultSchedulerConfig creates a SchedulerConfig with default values
func NewDefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		MentionCheckInterval: DefaultMentionCheckInterval,
		Tasks:                make(map[string]TaskConfig),
	}
}

// AddTask adds a new scheduled task configuration
func (c *SchedulerConfig) AddTask(name string, interval time.Duration) error {
	if interval < MinMentionCheckInterval || interval > MaxMentionCheckInterval {
		return fmt.Errorf("interval must be between %v and %v", MinMentionCheckInterval, MaxMentionCheckInterval)
	}

	c.Tasks[name] = TaskConfig{
		Interval: interval,
		Name:     name,
		Enabled:  true,
	}
	return nil
}
