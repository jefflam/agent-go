package scraper

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// Package scraper provides functionality for scraping Twitter data based on configured queries.
// It handles concurrent scraping with configurable workers, retries, and status reporting.

// Default configuration values
const (
	// DefaultWorkerCount defines the default number of concurrent scraping workers
	DefaultWorkerCount = 5

	// DefaultMaxRetries defines the default number of retry attempts for failed requests
	DefaultMaxRetries = 5

	// DefaultRetryBackoffMs defines the default backoff duration between retries in milliseconds
	DefaultRetryBackoffMs = 1000

	// DefaultStatusInterval defines how often the scraper reports its status
	DefaultStatusInterval = 30 * time.Second
)

// QueryConfig represents a single search query configuration with time bounds.
// It defines what data should be scraped and for what time period.
type QueryConfig struct {
	// Query is the search term or filter to be used
	Query string `json:"query"`

	// Count specifies the number of results to retrieve
	Count int `json:"count"`

	// StartDate defines the beginning of the time range to scrape
	StartDate time.Time `json:"startDate"`

	// EndDate defines the end of the time range to scrape
	EndDate time.Time `json:"endDate"`
}

// Config holds the complete configuration for the scraper including
// multiple queries and operational parameters.
type Config struct {
	// Queries contains the list of search queries to process
	Queries []QueryConfig `json:"queries"`

	// WorkerCount is the number of concurrent workers
	WorkerCount int

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryBackoffMs is the base retry backoff in milliseconds
	RetryBackoffMs int
}

// ScraperConfig represents the runtime configuration of the scraper,
// including the tasks to be processed and operational parameters.
type ScraperConfig struct {
	// Tasks is the list of individual scraping tasks to be processed
	Tasks []Task `json:"tasks"`

	// MaxRetries is the maximum number of retry attempts for failed requests
	MaxRetries int `json:"maxRetries"`

	// RetryBackoffMs is the duration to wait between retries in milliseconds
	RetryBackoffMs int `json:"retryBackoffMs"`

	// WorkerCount is the number of concurrent scraping workers
	WorkerCount int `json:"workerCount"`

	// StatusInterval defines how often the scraper should report its status
	StatusInterval time.Duration `json:"statusInterval"`
}

// LoadConfig reads and parses a configuration file from the given path.
// It returns a ScraperConfig containing the parsed configuration and any tasks
// generated from the query specifications.
//
// The configuration file should be in JSON format with the following structure:
//
//	{
//	    "queries": [
//	        {
//	            "query": "search term",
//	            "count": 100,
//	            "startDate": "2024-01-01",
//	            "endDate": "2024-01-31"
//	        }
//	    ]
//	}
//
// Each query will be split into daily tasks for processing.
func LoadConfig(path string) (*ScraperConfig, error) {
	type rawQuery struct {
		Query     string `json:"query"`
		Count     int    `json:"count"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	}

	type rawConfig struct {
		Queries []rawQuery `json:"queries"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	config := &ScraperConfig{
		MaxRetries:     DefaultMaxRetries,
		RetryBackoffMs: DefaultRetryBackoffMs,
		WorkerCount:    DefaultWorkerCount,
		StatusInterval: DefaultStatusInterval,
	}

	for _, q := range raw.Queries {
		startDate, err := time.Parse("2006-01-02", q.StartDate)
		if err != nil {
			return nil, fmt.Errorf("parsing start date: %w", err)
		}

		endDate, err := time.Parse("2006-01-02", q.EndDate)
		if err != nil {
			return nil, fmt.Errorf("parsing end date: %w", err)
		}

		// Split date range into daily tasks
		currentDate := startDate
		for currentDate.Before(endDate) || currentDate.Equal(endDate) {
			nextDate := currentDate.AddDate(0, 0, 1)
			task := Task{
				ID:        uuid.New().String(),
				Query:     q.Query,
				Count:     q.Count,
				StartDate: currentDate,
				EndDate:   nextDate,
				Status:    TaskStatusPending,
			}
			config.Tasks = append(config.Tasks, task)
			currentDate = nextDate
		}
	}

	return config, nil
}
