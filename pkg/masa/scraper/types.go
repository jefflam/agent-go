package scraper

import "time"

// ScraperStatus represents the current state of the scraper system,
// including task statistics and runtime information.
type ScraperStatus struct {
	// TotalTasks is the total number of tasks registered in the system
	TotalTasks int
	// CompletedTasks is the number of successfully completed tasks
	CompletedTasks int
	// FailedTasks is the number of tasks that failed and exceeded retry attempts
	FailedTasks int
	// RetryingTasks is the number of tasks currently in retry state
	RetryingTasks int
	// StartTime is when the scraper system was initialized
	StartTime time.Time
}

// TaskStatus represents the current state of a scraping task.
type TaskStatus string

// Task status constants define the possible states of a scraping task.
const (
	// TaskStatusPending indicates the task is queued but not yet started
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning indicates the task is currently executing
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusComplete indicates the task has finished successfully
	TaskStatusComplete TaskStatus = "complete"
	// TaskStatusFailed indicates the task has failed and won't be retried
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusRetrying indicates the task failed but will be retried
	TaskStatusRetrying TaskStatus = "retrying"
)

// Task represents a single Twitter data collection job with its parameters
// and current execution status.
type Task struct {
	// ID uniquely identifies the task
	ID string `json:"id"`
	// Query is the Twitter search query to execute
	Query string `json:"query"`
	// Count is the number of tweets to retrieve
	Count int `json:"count"`
	// StartDate defines the beginning of the time range to search
	StartDate time.Time `json:"startDate"`
	// EndDate defines the end of the time range to search
	EndDate time.Time `json:"endDate"`
	// Status indicates the current state of the task
	Status TaskStatus `json:"status"`
	// RetryCount tracks how many times this task has been retried
	RetryCount int `json:"retryCount"`
	// LastError contains the error message from the most recent failure
	LastError string `json:"lastError,omitempty"`
	// LastAttempt records when the task was last attempted
	LastAttempt time.Time `json:"lastAttempt,omitempty"`
}
