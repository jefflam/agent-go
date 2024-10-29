// Package twitter provides a client for interacting with the Twitter API v2.
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

// MentionResponse wraps TweetResponse with conversation tracking
type MentionResponse struct {
	Tweet          *TweetResponse
	ConversationID string
}

// GetUserMentions retrieves tweets mentioning a specific user.
// It returns two channels:
// - A channel that streams MentionResponse objects containing the mention data and conversation ID
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
func (c *TwitterClient) GetUserMentions(ctx context.Context, params GetUserMentionsParams) (chan *MentionResponse, chan error) {
	dataChan := make(chan *MentionResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		userID := params.UserID
		if userID == "" {
			// Get actual user ID instead of using "me"
			var err error
			userID, err = c.GetAuthenticatedUserID(ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get authenticated user ID: %w", err)
				return
			}
		}

		log := c.logger.WithFields(logrus.Fields{
			"method": "GetUserMentions",
			"userID": userID,
		})

		endpoint := fmt.Sprintf("/users/%s/mentions", userID)

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
						"author_id",
					), ","),
					"expansions": strings.Join(append(
						c.config.GetExpansions(),
						"referenced_tweets.id",
						"in_reply_to_user_id",
						"author_id",
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

				tweets, err := tweetResp.UnmarshalTweets()
				if err != nil {
					errChan <- fmt.Errorf("failed to unmarshal tweets: %w", err)
					return
				}

				// Create MentionResponse with conversation ID
				mentionResp := &MentionResponse{
					Tweet: &tweetResp,
				}

				// Extract conversation ID from the first tweet if available
				if len(tweets) > 0 {
					tweet := tweets[0]

					// If conversation_id is empty, use the tweet's ID as the conversation starter
					if tweet.ConversationID == "" {
						mentionResp.ConversationID = tweet.ID
					} else {
						mentionResp.ConversationID = tweet.ConversationID
					}

					c.logger.WithFields(logrus.Fields{
						"tweet_id":            tweet.ID,
						"conversation_id":     mentionResp.ConversationID,
						"is_new_conversation": tweet.ConversationID == "",
					}).Debug("Processing tweet conversation")
				}

				dataChan <- mentionResp

				if tweetResp.Meta.NextToken == "" {
					return
				}

				params.PaginationToken = tweetResp.Meta.NextToken
			}
		}
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
