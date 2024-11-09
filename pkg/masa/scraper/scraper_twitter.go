package scraper

import (
	"fmt"
	"sync"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/masa/masatwitter"
	"github.com/sirupsen/logrus"
)

// Package scraper provides functionality for scraping Twitter data using concurrent workers
// and handling retries with exponential backoff. It supports configurable batch processing
// of Twitter search queries with status reporting and error handling.

// Scraper handles Twitter data scraping with concurrent workers and retry logic.
// It maintains internal state of tasks and provides status reporting.
type Scraper struct {
	client    *masatwitter.Client
	logger    *logrus.Logger
	processor *TweetProcessor
	tasks     map[string]*Task
	status    ScraperStatus
	mu        sync.RWMutex
}

// NewScraper creates a new Scraper instance with the provided Twitter client and logger.
// It initializes the scraper with empty task map and zero status.
func NewScraper(client *masatwitter.Client, logger *logrus.Logger) *Scraper {
	logger.WithFields(logrus.Fields{
		"client": fmt.Sprintf("%T", client),
	}).Debug("Creating new Scraper instance")
	return &Scraper{
		client:    client,
		logger:    logger,
		processor: NewTweetProcessor(logger),
	}
}

// ProcessTasks executes the scraping tasks defined in the provided configuration.
// It manages concurrent workers, handles retries with exponential backoff, and
// provides periodic status updates. Returns an error if the processing fails.
//
// The function will:
// - Initialize worker pools based on config.WorkerCount
// - Process tasks concurrently with retry logic
// - Report status at config.StatusInterval intervals
// - Handle task completion and failure states
func (s *Scraper) ProcessTasks(config *ScraperConfig) error {
	s.mu.Lock()
	s.status = ScraperStatus{
		TotalTasks: len(config.Tasks),
		StartTime:  time.Now(),
	}

	// Initialize task map
	s.tasks = make(map[string]*Task)
	for _, task := range config.Tasks {
		s.tasks[task.ID] = &task
	}
	s.mu.Unlock()

	taskCh := make(chan *Task, len(config.Tasks))
	resultCh := make(chan *Task, len(config.Tasks))
	stopReporter := make(chan bool)
	done := make(chan struct{})

	// Start status reporter
	go s.reportStatus(config.StatusInterval, stopReporter)

	// Start result processor
	go func() {
		defer close(done)
		for task := range resultCh {
			s.mu.Lock()
			s.tasks[task.ID] = task
			switch task.Status {
			case TaskStatusComplete:
				s.status.CompletedTasks++
				s.status.RetryingTasks = max(0, s.status.RetryingTasks-1)
				s.logger.WithFields(logrus.Fields{
					"task_id":   task.ID,
					"completed": s.status.CompletedTasks,
					"total":     s.status.TotalTasks,
				}).Debug("Task completed")
			case TaskStatusFailed:
				if task.RetryCount < config.MaxRetries {
					task.Status = TaskStatusRetrying
					task.RetryCount++
					s.status.RetryingTasks++
					backoff := calculateBackoff(task.RetryCount, config.RetryBackoffMs)
					s.logger.WithFields(logrus.Fields{
						"task_id": task.ID,
						"retry":   task.RetryCount,
						"backoff": backoff.String(),
					}).Info("Scheduling task retry")
					time.AfterFunc(backoff, func() {
						taskCh <- task
					})
				} else {
					s.status.FailedTasks++
					s.status.RetryingTasks = max(0, s.status.RetryingTasks-1)
				}
			}
			s.mu.Unlock()

			// Check if all tasks are complete
			if s.status.CompletedTasks+s.status.FailedTasks == s.status.TotalTasks {
				close(stopReporter)
				return
			}
		}
	}()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < config.WorkerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.worker(id, taskCh, resultCh)
		}(i)
	}

	// Queue initial tasks
	for _, task := range config.Tasks {
		taskCopy := task
		taskCh <- &taskCopy
	}

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(taskCh)
	}()

	// Wait for result processing to complete
	<-done
	close(resultCh)

	return nil
}

// calculateBackoff determines the retry delay duration using exponential backoff.
// It ensures the backoff duration stays within defined minimum and maximum bounds.
func calculateBackoff(retryCount, baseBackoffMs int) time.Duration {
	const (
		minBackoff = 100 * time.Millisecond
		maxBackoff = 30 * time.Second
	)

	backoff := time.Duration(baseBackoffMs) *
		time.Millisecond * time.Duration(1<<retryCount)

	if backoff < minBackoff {
		return minBackoff
	}
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

// reportStatus periodically logs the current scraping status at specified intervals.
// It continues until receiving a signal on the stop channel.
func (s *Scraper) reportStatus(interval time.Duration, stop chan bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			s.logger.WithFields(logrus.Fields{
				"total":     s.status.TotalTasks,
				"completed": s.status.CompletedTasks,
				"failed":    s.status.FailedTasks,
				"retrying":  s.status.RetryingTasks,
				"duration":  time.Since(s.status.StartTime).String(),
			}).Info("Scraper status update")
			s.mu.RUnlock()
		case <-stop:
			return
		}
	}
}

// worker processes tasks from the input channel and sends results to the output channel.
// It handles task execution, error handling, and logging of task progress.
func (s *Scraper) worker(id int, tasks <-chan *Task, results chan<- *Task) {
	s.logger.WithField("worker_id", id).Debug("Worker started")

	for task := range tasks {
		s.logger.WithFields(logrus.Fields{
			"worker_id":   id,
			"task_id":     task.ID,
			"query":       task.Query,
			"retry_count": task.RetryCount,
		}).Debug("Processing task")

		queryStr := fmt.Sprintf("%s until:%s since:%s",
			task.Query,
			task.EndDate.Format("2006-01-02"),
			task.StartDate.Format("2006-01-02"),
		)

		tweets, err := s.client.SearchWithOptions(queryStr, masatwitter.SearchOptions{
			TweetCount: task.Count,
		})

		if err != nil {
			task.LastError = err.Error()
			task.LastAttempt = time.Now()
			task.Status = TaskStatusFailed

			s.logger.WithFields(logrus.Fields{
				"worker_id": id,
				"task_id":   task.ID,
				"error":     err,
				"retries":   task.RetryCount,
			}).Error("Task failed")

			results <- task
			continue
		}

		// Process tweets and mark task as complete
		s.processor.ProcessTweets(tweets)

		task.Status = TaskStatusComplete
		task.LastAttempt = time.Now()

		s.logger.WithFields(logrus.Fields{
			"worker_id": id,
			"task_id":   task.ID,
			"tweets":    len(tweets),
		}).Debug("Task completed successfully")

		results <- task
	}
}

// Status provides thread-safe access to the current scraper status through a callback function.
// The callback is executed while holding a read lock on the scraper's mutex.
func (s *Scraper) Status(callback func(ScraperStatus)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	callback(s.status)
}

// GetStatus returns a copy of the current scraper status in a thread-safe manner.
func (s *Scraper) GetStatus() ScraperStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

/*
// Example usage of the Twitter scraper:
// 1. Load configuration from JSON file

config, err := scraper.LoadConfig("pkg/masa/scraper/list.json")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// 2. Create new scraper instance with Twitter client and logger

scraper := NewScraper(client, logger)

// 3. Process tasks defined in config

if err := scraper.ProcessTasks(config); err != nil {
    log.Fatalf("Failed to process tasks: %v", err)
}
*/
