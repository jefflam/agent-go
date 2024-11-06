// Package masatwitter provides functionality for interacting with the Masa Protocol Twitter API
// and managing Twitter-related configurations.
package masatwitter

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Default configuration values
const (
	// DefaultAPIEndpoint is the default URL for the Masa Twitter API endpoint
	DefaultAPIEndpoint = "http://localhost:8080/api/v1/data/twitter/tweets/recent"
	// DefaultRequestTimeout is the default timeout in seconds for API requests
	DefaultRequestTimeout = 120
	// DefaultTweetsPerRequest is the default number of tweets to fetch per request
	DefaultTweetsPerRequest = 5
)

// Config holds the Masa Twitter API configuration settings.
// Environment variables:
//   - MASA_TWITTER_API_ENDPOINT: API endpoint URL (default: http://localhost:8080/api/v1/data/twitter/tweets/recent)
//   - MASA_TWITTER_REQUEST_TIMEOUT: Request timeout in seconds (default: 120)
//   - MASA_TWITTER_TWEETS_PER_REQUEST: Number of tweets per request (default: 5)
type Config struct {
	// APIEndpoint is the URL for the Masa Twitter API
	APIEndpoint string
	// RequestTimeout is the duration to wait before timing out requests
	RequestTimeout time.Duration
	// TweetsPerRequest is the maximum number of tweets to fetch per request
	TweetsPerRequest int
	// Logger is the configured logrus logger instance
	Logger *logrus.Logger
}

// NewConfig creates a new Config instance with values from environment variables.
// It loads configuration from environment variables and falls back to default values
// if not specified. The .env file is loaded if present, but its absence is not an error.
func NewConfig() (*Config, error) {
	logrus.Debug("Starting NewConfig initialization")

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			logrus.WithError(err).Error("Failed to load .env file")
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
		logrus.Debug(".env file not found, continuing with environment variables")
	} else {
		logrus.Debug("Successfully loaded .env file")
	}

	timeoutStr := os.Getenv("MASA_TWITTER_REQUEST_TIMEOUT")
	logrus.WithField("timeout_str", timeoutStr).Debug("Retrieved timeout from environment")

	timeout := DefaultRequestTimeout
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil {
			timeout = t
			logrus.WithField("timeout", timeout).Debug("Successfully parsed custom timeout")
		} else {
			logrus.WithFields(logrus.Fields{
				"value":   timeoutStr,
				"error":   err.Error(),
				"default": DefaultRequestTimeout,
			}).Debug("Failed to parse request timeout, using default")
		}
	} else {
		logrus.WithField("default_timeout", DefaultRequestTimeout).Debug("Using default timeout")
	}

	tweetsPerReq := DefaultTweetsPerRequest
	count := os.Getenv("MASA_TWITTER_TWEETS_PER_REQUEST")
	logrus.WithField("tweets_per_request_str", count).Debug("Retrieved tweets per request from environment")

	if count != "" {
		if t, err := strconv.Atoi(count); err == nil {
			tweetsPerReq = t
			logrus.WithField("tweets_per_request", tweetsPerReq).Debug("Successfully parsed custom tweets per request")
		} else {
			logrus.WithFields(logrus.Fields{
				"value":   count,
				"error":   err.Error(),
				"default": DefaultTweetsPerRequest,
			}).Debug("Failed to parse tweets per request, using default")
		}
	} else {
		logrus.WithField("default_tweets_per_request", DefaultTweetsPerRequest).Debug("Using default tweets per request")
	}

	apiEndpoint := getEnvOrDefault("MASA_TWITTER_API_ENDPOINT", DefaultAPIEndpoint)
	logrus.WithFields(logrus.Fields{
		"api_endpoint": apiEndpoint,
		"is_default":   apiEndpoint == DefaultAPIEndpoint,
	}).Debug("Retrieved API endpoint")

	logrus.WithFields(logrus.Fields{
		"api_endpoint":       apiEndpoint,
		"request_timeout":    timeout,
		"tweets_per_request": tweetsPerReq,
	}).Debug("Creating new Masa Twitter config")

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logrus.Debug("Created new logger instance with debug level")

	config := &Config{
		APIEndpoint:      apiEndpoint,
		RequestTimeout:   time.Duration(timeout) * time.Second,
		TweetsPerRequest: tweetsPerReq,
		Logger:           logger,
	}

	logrus.WithFields(logrus.Fields{
		"api_endpoint":       config.APIEndpoint,
		"request_timeout":    config.RequestTimeout.String(),
		"tweets_per_request": config.TweetsPerRequest,
		"logger_level":       config.Logger.GetLevel().String(),
	}).Debug("Created config instance")

	if err := config.Validate(); err != nil {
		logrus.WithError(err).Debug("Config validation failed")
		return nil, err
	}

	logrus.Debug("Successfully created and validated Masa Twitter config")
	return config, nil
}

// Validate checks if the configuration is valid according to the following rules:
//   - APIEndpoint must not be empty
//   - Logger must be initialized
//   - RequestTimeout must be at least 1 second
//   - TweetsPerRequest must be positive
func (c *Config) Validate() error {
	logrus.Debug("Starting config validation")

	logrus.WithFields(logrus.Fields{
		"api_endpoint":       c.APIEndpoint,
		"request_timeout":    c.RequestTimeout.String(),
		"tweets_per_request": c.TweetsPerRequest,
		"logger":             c.Logger != nil,
	}).Debug("Validating config fields")

	if c.APIEndpoint == "" {
		logrus.Error("API endpoint validation failed: empty endpoint")
		return fmt.Errorf("masatwitter: API endpoint is required")
	}
	logrus.Debug("API endpoint validation passed")

	if c.Logger == nil {
		logrus.Error("Logger validation failed: nil logger")
		return fmt.Errorf("masatwitter: logger is required")
	}
	logrus.Debug("Logger validation passed")

	if c.RequestTimeout < 1*time.Second {
		logrus.WithField("timeout", c.RequestTimeout).Error("Request timeout validation failed: too short")
		return fmt.Errorf("masatwitter: request timeout must be at least 1 second, got %v", c.RequestTimeout)
	}
	logrus.Debug("Request timeout validation passed")

	if c.TweetsPerRequest < 1 {
		logrus.WithField("tweets_per_request", c.TweetsPerRequest).Error("Tweets per request validation failed: non-positive value")
		return fmt.Errorf("masatwitter: tweets per request must be positive, got %d", c.TweetsPerRequest)
	}
	logrus.Debug("Tweets per request validation passed")

	logrus.Debug("Config validation successful")
	return nil
}

// getEnvOrDefault retrieves an environment variable value by key,
// returning the defaultValue if the environment variable is not set or empty.
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	logrus.WithFields(logrus.Fields{
		"key":           key,
		"value":         value,
		"default":       defaultValue,
		"using_default": value == "",
	}).Debug("Getting environment variable")

	if value != "" {
		return value
	}
	return defaultValue
}
