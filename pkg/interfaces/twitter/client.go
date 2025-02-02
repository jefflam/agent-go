package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ClientOption allows for customization of the client
type ClientOption func(*TwitterClient)

type TwitterClient struct {
	config *TwitterConfig
	auth   *Authenticator
	logger *logrus.Logger
	log    *logrus.Logger
}

// NewTwitterClient creates a new Twitter API client
func NewTwitterClient(config *TwitterConfig, opts ...ClientOption) (*TwitterClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	auth, err := NewAuthenticator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	client := &TwitterClient{
		config: config,
		auth:   auth,
		logger: config.Logger,
		log:    logrus.New(),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// handleResponse checks for API errors in the response
func (c *TwitterClient) handleResponse(resp *http.Response) error {
	// Log response headers and status
	c.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
	}).Debug("Received API response")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read error response body")
		return fmt.Errorf("failed to read error response: %w", err)
	}

	// Log raw response body
	c.logger.WithField("response_body", string(body)).Debug("Error response body")

	var errResp struct {
		Errors []struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("twitter api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	if len(errResp.Errors) > 0 {
		c.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"error_code":  errResp.Errors[0].Code,
			"message":     errResp.Errors[0].Message,
		}).Error("Twitter API error")
		return fmt.Errorf("twitter api error: code=%d message=%s",
			errResp.Errors[0].Code, errResp.Errors[0].Message)
	}

	return fmt.Errorf("twitter api error: status=%d", resp.StatusCode)
}

func (c *TwitterClient) handleRateLimits(resp *http.Response) error {
	// Only handle 429 responses
	if resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}

	// Log all relevant rate limit headers
	c.logger.WithFields(logrus.Fields{
		"endpoint":                      resp.Request.URL.Path,
		"x_rate_limit_limit":            resp.Header.Get("x-rate-limit-limit"),
		"x_rate_limit_remaining":        resp.Header.Get("x-rate-limit-remaining"),
		"x_rate_limit_reset":            resp.Header.Get("x-rate-limit-reset"),
		"x_user_limit_24hour":           resp.Header.Get("x-user-limit-24hour"),
		"x_user_limit_24hour_remaining": resp.Header.Get("x-user-limit-24hour-remaining"),
		"x_user_limit_24hour_reset":     resp.Header.Get("x-user-limit-24hour-reset"),
	}).Debug("Rate limit headers received")

	// Get endpoint-specific and daily limits
	endpointRemaining := parseIntHeader(resp.Header.Get("x-rate-limit-remaining"))
	endpointReset := parseInt64Header(resp.Header.Get("x-rate-limit-reset"))
	dailyRemaining := parseIntHeader(resp.Header.Get("x-user-limit-24hour-remaining"))
	dailyReset := parseInt64Header(resp.Header.Get("x-user-limit-24hour-reset"))

	// Use the more restrictive reset time
	var resetTime time.Time
	if endpointReset > dailyReset {
		resetTime = time.Unix(endpointReset, 0)
	} else {
		resetTime = time.Unix(dailyReset, 0)
	}

	waitDuration := time.Until(resetTime)

	c.logger.WithFields(logrus.Fields{
		"endpoint_remaining": endpointRemaining,
		"daily_remaining":    dailyRemaining,
		"reset_time":         resetTime.Format(time.RFC3339),
		"wait_duration":      waitDuration.Round(time.Second),
	}).Warning("Rate limit exceeded")

	return fmt.Errorf("rate limit exceeded, reset in %v at %v",
		waitDuration.Round(time.Second),
		resetTime.Format(time.RFC3339))
}

func (c *TwitterClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	c.logger.WithFields(logrus.Fields{
		"method":   method,
		"endpoint": endpoint,
	}).Debug("Preparing API request")

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			c.logger.WithError(err).Error("Failed to marshal request body")
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
		c.logger.WithFields(logrus.Fields{
			"request_body": string(jsonBody),
			"content_type": "application/json",
		}).Debug("Request payload")
	}

	fullURL := c.config.BaseURL + endpoint
	c.logger.WithField("url", fullURL).Debug("Making request to Twitter API")

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log request headers
	c.logger.WithFields(logrus.Fields{
		"headers": req.Header,
	}).Debug("Request headers")

	// OAuth 1.0a client will handle the authentication headers
	resp, err := c.auth.GetClient().Do(req)
	if err != nil {
		c.logger.WithError(err).Error("Request failed")
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Log rate limit headers
	c.logger.WithFields(logrus.Fields{
		"endpoint":               endpoint,
		"x-rate-limit-limit":     resp.Header.Get("x-rate-limit-limit"),
		"x-rate-limit-remaining": resp.Header.Get("x-rate-limit-remaining"),
		"x-rate-limit-reset":     resp.Header.Get("x-rate-limit-reset"),
	}).Debug("Rate limit headers")

	if resp.StatusCode == http.StatusTooManyRequests {
		if err := c.handleRateLimits(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}

	// Add error handling here
	if err := c.handleResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

// makeRequestWithParams makes a request to the Twitter API with query parameters
func (c *TwitterClient) makeRequestWithParams(ctx context.Context, method, endpoint string, queryParams map[string]string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.config.BaseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	// Add OAuth 1.0a authentication for endpoints requiring user context
	if strings.Contains(endpoint, "/mentions") {
		authHeader, err := c.generateOAuth1Header(method, req.URL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to generate OAuth header: %w", err)
		}
		req.Header.Set("Authorization", authHeader)
	} else {
		// Use Bearer token for other endpoints
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.BearerToken))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":    req.URL.String(),
		"method": method,
		"params": queryParams,
	}).Debug("Making request to Twitter API")

	resp, err := c.auth.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if err := c.handleRateLimits(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("twitter API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// Helper functions to parse headers
func parseIntHeader(value string) int {
	if value == "" {
		return 0
	}
	i, _ := strconv.Atoi(value)
	return i
}

func parseInt64Header(value string) int64 {
	if value == "" {
		return 0
	}
	i, _ := strconv.ParseInt(value, 10, 64)
	return i
}
