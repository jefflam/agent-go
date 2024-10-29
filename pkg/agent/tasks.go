package agent

import (
	"time"
)

// Default intervals for different tasks
const (
	DefaultMentionInterval   = 60 * time.Second
	DefaultTimelineInterval  = 5 * time.Minute
	DefaultHeartbeatInterval = 30 * time.Second
)

// Task Types
const (
	TaskMentions TaskType = "mentions" // Handles @mentions
)

// TaskPriority defines the importance level of a task
type TaskPriority int

const (
	PriorityLow TaskPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// TaskMetadata holds additional information about a task
type TaskMetadata struct {
	Description  string
	Priority     TaskPriority
	Dependencies []TaskType
}

// DefaultTaskConfigs provides the default configuration for all supported tasks
var DefaultTaskConfigs = map[TaskType]TaskConfig{
	TaskMentions: {
		Enabled:  true,
		Interval: DefaultMentionInterval,
		Metadata: TaskMetadata{
			Description: "Processes and responds to @mentions",
			Priority:    PriorityHigh,
		},
	},
}

// TaskManager provides methods for task management
type TaskManager interface {
	EnableTask(taskType TaskType) error
	DisableTask(taskType TaskType) error
	GetTaskStatus(taskType TaskType) (bool, error)
	GetTaskMetadata(taskType TaskType) (TaskMetadata, error)
	ListActiveTasks() []TaskType
}

// TaskValidator validates task configurations
type TaskValidator interface {
	ValidateConfig(config TaskConfig) error
	ValidateDependencies(taskType TaskType, configs map[TaskType]TaskConfig) error
}

// Helper functions for task management
func IsTaskEnabled(configs map[TaskType]TaskConfig, taskType TaskType) bool {
	if config, exists := configs[taskType]; exists {
		return config.Enabled
	}
	return false
}

func GetTaskInterval(configs map[TaskType]TaskConfig, taskType TaskType) time.Duration {
	if config, exists := configs[taskType]; exists {
		return config.Interval
	}
	// Return a safe default
	return 5 * time.Minute
}

func ValidateTaskConfigs(configs map[TaskType]TaskConfig) error {
	// Implement validation logic here
	// - Check for required tasks
	// - Validate intervals
	// - Check dependencies
	// - Ensure no conflicts
	return nil
}
