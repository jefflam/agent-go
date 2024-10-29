// Package twitter provides a client for interacting with the Twitter API v2.
package twitter

import (
	"context"
	"encoding/json"
	"fmt"

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
				// Create mention response
				mentionResp := &MentionResponse{}

				resp, err := c.makeRequest(ctx, "GET", endpoint, params)
				if err != nil {
					log.WithError(err).Error("failed to fetch user mentions")
					errChan <- err
					return
				}
				defer resp.Body.Close()

				if err := json.NewDecoder(resp.Body).Decode(mentionResp); err != nil {
					log.WithError(err).Error("failed to decode response")
					errChan <- err
					return
				}

				// Log conversation details
				if len(mentionResp.Data) > 0 {
					c.logger.WithFields(logrus.Fields{
						"tweet_id":        mentionResp.Data[0].ID,
						"conversation_id": mentionResp.Data[0].ConversationID,
					}).Debug("Processing tweet conversation")
				}

				dataChan <- mentionResp

				if mentionResp.Meta.NextToken == "" {
					return
				}

				params.PaginationToken = mentionResp.Meta.NextToken
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
