package integration

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

func TestGetUserMentions(t *testing.T) {
	// Skip if not running integration tests
	if integrationTests := os.Getenv("INTEGRATION_TESTS"); integrationTests != "true" {
		t.Skip("Skipping integration test")
	}

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Get required environment variables
	bearerToken := os.Getenv("TWITTER_BEARER_TOKEN")
	if bearerToken == "" {
		t.Fatal("TWITTER_BEARER_TOKEN environment variable is required")
	}

	// Initialize config
	config := &twitter.TwitterConfig{
		BearerToken:       os.Getenv("TWITTER_BEARER_TOKEN"),
		ConsumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
		AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
		BaseURL:           "https://api.twitter.com/2", // Ensure this doesn't have a trailing slash
		RateLimit:         180,
		RateWindow:        int(15 * time.Minute / time.Second),
		Logger:            logger,
		DefaultFields:     []string{"id", "text", "created_at"},
		MetricFields:      []string{"like_count", "reply_count", "retweet_count"},
		ExpansionFields: []string{
			"author_id",
			"referenced_tweets.id",
			"in_reply_to_user_id",
		},
	}

	client, err := twitter.NewTwitterClient(config)
	require.NoError(t, err)

	// Get authenticated user ID first
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID, err := client.GetAuthenticatedUserID(ctx)
	require.NoError(t, err, "Failed to get authenticated user ID")

	tests := []struct {
		name          string
		params        twitter.GetUserMentionsParams
		expectedError bool
		errorContains string
	}{
		{
			name: "Get authenticated user mentions",
			params: twitter.GetUserMentionsParams{
				MaxResults: 10,
			},
			expectedError: false,
		},
		{
			name: "Invalid max results should return error",
			params: twitter.GetUserMentionsParams{
				MaxResults: 101,
			},
			expectedError: true,
			errorContains: "invalid max_results",
		},
		{
			name: "Get mentions with specific user ID",
			params: twitter.GetUserMentionsParams{
				UserID:     userID, // Use actual user ID instead of env var
				MaxResults: 10,
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dataChan, errChan := client.GetUserMentions(ctx, tt.params)

			// Handle response
			var receivedData bool
			var lastError error

			for {
				select {
				case resp, ok := <-dataChan:
					if !ok {
						dataChan = nil
						continue
					}
					receivedData = true
					if resp != nil {
						// Validate response structure
						assert.NotNil(t, resp.Meta)
						if len(resp.Data) > 0 {
							assert.NotEmpty(t, resp.Data[0].ID)
							assert.NotEmpty(t, resp.Data[0].Text)

							// Log the mention for debugging
							t.Logf("Received mention: %s", resp.Data[0].Text)
						}
					}
				case err, ok := <-errChan:
					if !ok {
						errChan = nil
						continue
					}
					lastError = err
					t.Logf("Received error: %v", err)
				case <-ctx.Done():
					t.Logf("Context deadline exceeded")
					return
				}

				if dataChan == nil && errChan == nil {
					break
				}
			}

			// Add validation for 404 responses
			if lastError != nil && !tt.expectedError {
				if strings.Contains(lastError.Error(), "status=404") {
					t.Errorf("Received 404 error - check API endpoint URL and authentication: %v", lastError)
				}
			}

			// Improve error handling
			if tt.expectedError {
				require.Error(t, lastError)
				if tt.errorContains != "" {
					assert.Contains(t, lastError.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, lastError)
				assert.True(t, receivedData, "Should have received data")
			}
		})
	}
}
