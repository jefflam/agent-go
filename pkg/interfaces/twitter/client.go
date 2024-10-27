package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// ClientOption allows for customization of the client
type ClientOption func(*TwitterClient)

type TwitterClient struct {
	config *TwitterConfig
	auth   *Authenticator
	logger *logrus.Logger
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
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// handleResponse checks for API errors in the response
func (c *TwitterClient) handleResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

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

func (c *TwitterClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
		c.logger.WithField("request_body", string(jsonBody)).Debug("Request payload")
	}

	fullURL := c.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// OAuth 1.0a client will handle the authentication headers
	resp, err := c.auth.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}
