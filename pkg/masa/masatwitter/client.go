// Package masatwitter provides functionality for interacting with the Masa Protocol Twitter API
package masatwitter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Client handles Masa Twitter API interactions by providing methods to search and retrieve tweets.
// It manages configuration and HTTP client lifecycle.
type Client struct {
	config *Config
	client *http.Client
	logger *logrus.Logger
}

// SearchRequest represents the search query parameters sent to the API.
type SearchRequest struct {
	// Query is the search term or filter to apply
	Query string `json:"query"`
	// Count specifies the maximum number of tweets to return
	Count int `json:"count"`
}

// SearchOptions allows customizing the search request parameters.
type SearchOptions struct {
	// TweetCount specifies the maximum number of tweets to return
	TweetCount int
}

// NewClient creates a new Masa Twitter API client with the provided configuration.
// It initializes an HTTP client with the configured timeout.
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
		logger: config.Logger,
	}
}

// Search performs a search request to the Masa Twitter API with default options.
// It uses the configured default tweets per request count.
func (c *Client) Search(query string) ([]Tweet, error) {
	c.logger.WithFields(logrus.Fields{
		"query":              query,
		"tweets_per_request": c.config.TweetsPerRequest,
	}).Debug("Performing search with default options")

	return c.SearchWithOptions(query, SearchOptions{
		TweetCount: c.config.TweetsPerRequest,
	})
}

// SearchWithOptions performs a search request to the Masa Twitter API with custom options.
// It handles the full request lifecycle including:
// - Request marshaling
// - HTTP request creation and execution
// - Response handling and error processing
// - Rate limit detection
// - Response unmarshaling
func (c *Client) SearchWithOptions(query string, opts SearchOptions) ([]Tweet, error) {
	c.logger.WithFields(logrus.Fields{
		"query":      query,
		"tweetCount": opts.TweetCount,
	}).Debug("Starting search with custom options")

	reqBody := SearchRequest{
		Query: query,
		Count: opts.TweetCount,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal request body")
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	c.logger.WithField("request_body", string(jsonBody)).Debug("Marshaled request body")

	req, err := http.NewRequest("POST", c.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.logger.WithError(err).Error("Failed to create HTTP request")
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.WithFields(logrus.Fields{
		"endpoint": c.config.APIEndpoint,
		"method":   req.Method,
		"headers":  req.Header,
	}).Debug("Sending HTTP request")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("HTTP request failed")
		return nil, &ConnectionError{Err: err}
	}
	defer resp.Body.Close()

	c.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
	}).Debug("Received HTTP response")

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == StatusRateLimit {
			c.logger.Warn("Rate limit exceeded")
			return nil, NewRateLimitError(0, "")
		}
		c.logger.WithField("status_code", resp.StatusCode).Error("Unexpected status code")
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		}
	}

	// Keep the intermediate structure for unmarshaling
	var response struct {
		Data []struct {
			Tweet Tweet `json:"Tweet"`
		} `json:"data"`
		WorkerPeerID string `json:"workerPeerId"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.WithError(err).Error("Failed to decode response")
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Extract just the tweets
	tweets := make([]Tweet, len(response.Data))
	for i, item := range response.Data {
		tweets[i] = item.Tweet
	}

	c.logger.WithField("tweets_count", len(tweets)).Debug("Successfully decoded response")

	return tweets, nil
}
