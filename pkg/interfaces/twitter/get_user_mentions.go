// Package twitter provides a client for interacting with the Twitter API v2.
// It handles authentication, rate limiting, pagination and provides strongly typed
// responses for various Twitter endpoints.
//
// The package implements Twitter's best practices for:
// - Authentication: Supports both App-only and User authentication flows
// - Rate Limiting: Built-in rate limit handling and backoff strategies
// - Pagination: Cursor-based pagination via NextToken for efficient data retrieval
// - Error Handling: Structured error types with detailed error information
// - Response Types: Strongly typed response objects for type safety
//
// Rate Limits:
// - User Mentions: 450 requests/15min (app auth), 180 requests/15min (user auth)
// - User Tweets: 1500 requests/15min (app auth), 900 requests/15min (user auth)
// - User Lookup: 300 requests/15min (app auth), 900 requests/15min (user auth)
// - Tweet Lookup: 300 requests/15min (app auth), 900 requests/15min (user auth)
//
// The client automatically handles pagination and provides channels for streaming
// responses, making it easy to process large datasets efficiently.
package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetUserMentionsParams holds the parameters for the GetUserMentions request
type GetUserMentionsParams struct {
	// UserID specifies which user's mentions to retrieve.
	// If empty, defaults to the authenticated user.
	UserID string

	// PaginationToken is used for cursor-based pagination through results.
	// Leave empty for the first request.
	PaginationToken string

	// MaxResults specifies the number of tweets to return per page.
	// Valid values are 5-100. If not specified, defaults to 10.
	MaxResults int
}

// GetUserMentions retrieves tweets mentioning a specific user.
// It returns two channels:
// - A channel that streams TweetResponse objects containing the mention data
// - An error channel for any errors encountered during processing
//
// The endpoint uses cursor-based pagination via NextToken to retrieve all results.
// Results include:
// - Tweet text and metadata
// - Author information
// - Conversation threading details
// - Referenced tweet information
//
// Rate Limits:
// - App-only auth: 450 requests per 15-minute window
// - User auth: 180 requests per 15-minute window
//
// Example usage:
//
//	params := GetUserMentionsParams{
//	    MaxResults: 100,
//	}
//	dataChan, errChan := client.GetUserMentions(ctx, params)
//	for {
//	    select {
//	    case resp, ok := <-dataChan:
//	        if !ok {
//	            return // Channel closed
//	        }
//	        // Process tweets
//	    case err := <-errChan:
//	        log.Error(err)
//	        return
//	    }
//	}
func (c *TwitterClient) GetUserMentions(ctx context.Context, params GetUserMentionsParams) (chan *TweetResponse, chan error) {
	dataChan := make(chan *TweetResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		// Use authenticated user's ID if UserID is not provided
		userID := params.UserID
		if userID == "" {
			// Use "me" as a special identifier for the authenticated user
			userID = "me"
		}

		log := c.logger.WithFields(logrus.Fields{
			"method": "GetUserMentions",
			"userID": userID,
		})

		endpoint := fmt.Sprintf("%s/users/%s/mentions", c.config.BaseURL, userID)

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				body := map[string]interface{}{
					"pagination_token": params.PaginationToken,
					"max_results":      params.MaxResults,
					"tweet.fields": strings.Join(append(
						c.config.GetTweetFields(),
						"conversation_id",
						"in_reply_to_user_id",
						"referenced_tweets",
					), ","),
					"expansions": strings.Join(append(
						c.config.GetExpansions(),
						"referenced_tweets.id",
						"in_reply_to_user_id",
					), ","),
				}

				resp, err := c.makeRequest(ctx, "GET", endpoint, body)
				if err != nil {
					log.WithError(err).Error("failed to fetch user mentions")
					errChan <- err
					return
				}
				defer resp.Body.Close()

				var tweetResp TweetResponse
				if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
					log.WithError(err).Error("failed to decode response")
					errChan <- err
					return
				}

				dataChan <- &tweetResp

				if tweetResp.Meta.NextToken == "" {
					return
				}

				params.PaginationToken = tweetResp.Meta.NextToken
			}
		}
	}()

	return dataChan, errChan
}
