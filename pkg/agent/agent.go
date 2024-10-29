package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// New creates a new Agent instance
func New(config Config) (*Agent, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// If no tasks are configured, use default configurations
	if len(config.Tasks) == 0 {
		config.Tasks = DefaultTaskConfigs
	}

	agent := &Agent{
		client:      config.TwitterClient,
		llm:         config.LLM,
		logger:      config.Logger,
		tasks:       make(map[TaskType]Task),
		taskConfigs: config.Tasks,
	}

	// Initialize tasks
	if err := agent.initializeTasks(config.Tasks); err != nil {
		return nil, fmt.Errorf("failed to initialize tasks: %w", err)
	}

	return agent, nil
}

// Run starts all enabled agent tasks
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("Starting agent with all enabled tasks")

	var wg sync.WaitGroup
	errChan := make(chan error, len(a.tasks))

	// Start all tasks
	a.tasksMu.RLock()
	for taskType, task := range a.tasks {
		wg.Add(1)
		go func(t Task, tt TaskType) {
			defer wg.Done()
			a.logger.WithField("task", tt).Info("Starting task")

			if err := t.Run(ctx); err != nil && err != context.Canceled {
				a.logger.WithError(err).WithField("task", tt).Error("Task failed")
				errChan <- fmt.Errorf("task %s failed: %w", tt, err)
			}
		}(task, taskType)
	}
	a.tasksMu.RUnlock()

	// Block until context is canceled or a task returns an error
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	// Wait for either context cancellation or completion
	select {
	case <-ctx.Done():
		a.logger.Info("Context canceled, initiating shutdown")
		a.Stop() // Stop all tasks
		<-done   // Wait for all tasks to finish
		return ctx.Err()
	case err := <-errChan:
		a.logger.WithError(err).Error("Task error occurred")
		a.Stop() // Stop all tasks on error
		<-done   // Wait for all tasks to finish
		return err
	case <-done:
		a.logger.Info("All tasks completed normally")
		return nil
	}
}

// Stop stops all running tasks
func (a *Agent) Stop() {
	a.tasksMu.RLock()
	defer a.tasksMu.RUnlock()

	for taskType, task := range a.tasks {
		a.logger.WithField("task", taskType).Info("Stopping task")
		task.Stop()
	}
}

// AddTask adds a new task to the agent
func (a *Agent) AddTask(taskType TaskType, task Task) error {
	a.tasksMu.Lock()
	defer a.tasksMu.Unlock()

	if _, exists := a.tasks[taskType]; exists {
		return fmt.Errorf("task %s already exists", taskType)
	}

	a.tasks[taskType] = task
	return nil
}

// RemoveTask removes a task from the agent
func (a *Agent) RemoveTask(taskType TaskType) {
	a.tasksMu.Lock()
	defer a.tasksMu.Unlock()

	if task, exists := a.tasks[taskType]; exists {
		task.Stop()
		delete(a.tasks, taskType)
	}
}

func validateConfig(config Config) error {
	if config.LLM == nil {
		return fmt.Errorf("LLM is required")
	}
	if config.TwitterClient == nil {
		return fmt.Errorf("TwitterClient is required")
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}
	return nil
}

func (a *Agent) initializeTasks(taskConfigs map[TaskType]TaskConfig) error {
	for taskType, config := range taskConfigs {
		if !config.Enabled {
			continue
		}

		var task Task
		switch taskType {
		case TaskMentions:
			task = NewMentionProcessor(a.client, a.llm, a.logger, config.Interval)
		default:
			return fmt.Errorf("unknown task type: %s", taskType)
		}

		if err := a.AddTask(taskType, task); err != nil {
			return err
		}
	}
	return nil
}
