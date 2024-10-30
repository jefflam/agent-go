// Package twitter provides a client for interacting with the Twitter API v2.
package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetUserMentionsParams represents parameters for fetching user mentions
type GetUserMentionsParams struct {
	UserID          string   `json:"user_id,omitempty"`
	MaxResults      int      `json:"max_results,omitempty"`
	PaginationToken string   `json:"pagination_token,omitempty"`
	SinceID         string   `json:"since_id,omitempty"`
	UntilID         string   `json:"until_id,omitempty"`
	StartTime       string   `json:"start_time,omitempty"`
	EndTime         string   `json:"end_time,omitempty"`
	TweetFields     []string `json:"tweet.fields,omitempty"`
	UserFields      []string `json:"user.fields,omitempty"`
	Expansions      []string `json:"expansions,omitempty"`
	MediaFields     []string `json:"media.fields,omitempty"`
	PlaceFields     []string `json:"place.fields,omitempty"`
	PollFields      []string `json:"poll.fields,omitempty"`
}

// GetConversationID returns the conversation ID from the first tweet in the response
func (mr *MentionResponse) GetConversationID() string {
	if mr == nil || len(mr.Data) == 0 {
		return ""
	}
	return mr.Data[0].ConversationID
}

// Convert MentionResponse to TweetResponse for compatibility
func (mr *MentionResponse) ToTweetResponse() (*TweetResponse, error) {
	if mr == nil {
		return nil, nil
	}

	// Convert the tweets array to json.RawMessage
	rawData, err := json.Marshal(mr.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tweets: %w", err)
	}

	return &TweetResponse{
		Data:     json.RawMessage(rawData),
		Includes: mr.Includes,
		Errors:   mr.Errors,
		Meta:     mr.Meta,
	}, nil
}

// GetUserMentions retrieves tweets mentioning a specific user
func (c *TwitterClient) GetUserMentions(ctx context.Context, params GetUserMentionsParams) (<-chan *MentionResponse, <-chan error) {
	dataChan := make(chan *MentionResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		// Get authenticated user ID if not provided
		if params.UserID == "" {
			authUserID, err := c.GetAuthenticatedUserID(ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get authenticated user ID: %w", err)
				return
			}
			params.UserID = authUserID
		}

		// Validate MaxResults
		if params.MaxResults > 100 {
			errChan <- fmt.Errorf("invalid max_results: must be between 1 and 100")
			return
		}

		// Log the incoming params
		c.logger.WithFields(logrus.Fields{
			"incoming_params": params,
		}).Debug("Received GetUserMentions params")

		// Build query parameters with detailed logging
		queryParams := map[string]string{
			"max_results": fmt.Sprintf("%d", params.MaxResults),
		}

		// Add tweet fields with logging
		tweetFields := params.TweetFields
		if len(tweetFields) == 0 {
			tweetFields = []string{
				"id",
				"text",
				"created_at",
				"conversation_id",
				"in_reply_to_user_id",
				"referenced_tweets",
				"public_metrics",
				"author_id",
				"reply_settings",
			}
			c.logger.Debug("Using default tweet fields")
		}
		queryParams["tweet.fields"] = strings.Join(tweetFields, ",")

		// Add expansions with logging
		expansions := params.Expansions
		if len(expansions) == 0 {
			expansions = []string{
				"author_id",
				"referenced_tweets.id",
				"in_reply_to_user_id",
				"entities.mentions.username",
				"referenced_tweets.id.author_id",
			}
			c.logger.Debug("Using default expansions")
		}
		queryParams["expansions"] = strings.Join(expansions, ",")

		// Log the final query parameters
		c.logger.WithFields(logrus.Fields{
			"endpoint":     fmt.Sprintf("/users/%s/mentions", params.UserID),
			"tweet_fields": queryParams["tweet.fields"],
			"expansions":   queryParams["expansions"],
			"max_results":  queryParams["max_results"],
		}).Debug("Final API request parameters")

		endpoint := fmt.Sprintf("/users/%s/mentions", params.UserID)
		resp, err := c.makeRequestWithParams(ctx, "GET", endpoint, queryParams)
		if err != nil {
			errChan <- fmt.Errorf("failed to make request: %w", err)
			return
		}
		defer resp.Body.Close()

		var mentionResp MentionResponse
		if err := json.NewDecoder(resp.Body).Decode(&mentionResp); err != nil {
			errChan <- fmt.Errorf("failed to decode response: %w", err)
			return
		}

		dataChan <- &mentionResp
	}()

	return dataChan, errChan
}

// GetAuthenticatedUserID retrieves the authenticated user's ID
func (c *TwitterClient) GetAuthenticatedUserID(ctx context.Context) (string, error) {
	endpoint := "/users/me"
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var userResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", err
	}
	return userResp.Data.ID, nil
}
