// Package twitter provides a client for interacting with the Twitter API v2.
// It handles authentication, rate limiting, pagination and provides strongly typed
// responses for various Twitter endpoints.
//
// The package implements Twitter's best practices for:
// - Authentication: Supports both App-only and User authentication flows
// - Rate Limiting: Built-in rate limit handling and backoff strategies
// - Error Handling: Structured error types with detailed error information
// - Response Types: Strongly typed response objects for type safety
//
// Rate Limits:
// - Tweet Creation: 200 tweets/15min (user auth only)
// - Tweet Deletion: 50 tweets/15min (user auth only)
// - Tweet Lookup: 300 requests/15min (app auth), 900 requests/15min (user auth)
// - Media Upload: 30MB total/day
package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/sirupsen/logrus"
)

// TweetMedia represents media attachments in a tweet
// Media can include images, videos, and GIFs with the following limits:
// - Images: up to 4 images per tweet, max 5MB each
// - Video: 1 video per tweet, max 512MB
// - GIF: 1 GIF per tweet, max 15MB
type TweetMedia struct {
	MediaIDs []string `json:"media_ids"`
}

// BaseTweetRequest contains common fields for all tweet-related requests
// This struct is used as the foundation for creating, updating and managing tweets
type BaseTweetRequest struct {
	Text                  string      `json:"text"`                           // Tweet text content
	ReplySettings         string      `json:"reply_settings,omitempty"`       // Controls who can reply: "everyone", "mentionedUsers", "following"
	ForSuperFollowersOnly bool        `json:"for_super_followers_only"`       // Whether the tweet is for super followers only
	QuoteTweetID          string      `json:"quote_tweet_id,omitempty"`       // ID of tweet being quoted
	ReplyTo               string      `json:"in_reply_to_tweet_id,omitempty"` // ID of tweet being replied to
	ConversationID        string      `json:"conversation_id,omitempty"`      // ID of the conversation thread
	Poll                  *Poll       `json:"poll,omitempty"`                 // Optional poll attachment
	Media                 *TweetMedia `json:"media,omitempty"`                // Optional media attachments
}

// BaseOptions contains common optional parameters for tweet operations
// These options can be used to customize tweet creation and updates
type BaseOptions struct {
	ReplyTo               string  `json:"reply_to,omitempty"`        // ID of tweet to reply to
	QuoteTweetID          string  `json:"quote_tweet_id,omitempty"`  // ID of tweet to quote
	ConversationID        string  `json:"conversation_id,omitempty"` // Thread conversation ID
	Poll                  *Poll   `json:"poll,omitempty"`            // Poll configuration
	Media                 []Media `json:"media,omitempty"`           // Media attachments
	ReplySettings         string  `json:"reply_settings,omitempty"`  // Reply permission settings
	ForSuperFollowersOnly bool    `json:"for_super_followers_only"`  // Super followers only flag
}

// buildBaseRequest creates a BaseTweetRequest from text and options
// It handles the conversion of options into the appropriate request format
func buildBaseRequest(text string, opts *BaseOptions) BaseTweetRequest {
	request := BaseTweetRequest{
		Text: text,
	}

	if opts != nil {
		request.ReplySettings = opts.ReplySettings
		request.ForSuperFollowersOnly = opts.ForSuperFollowersOnly
		request.QuoteTweetID = opts.QuoteTweetID
		request.ReplyTo = opts.ReplyTo
		request.ConversationID = opts.ConversationID
		request.Poll = opts.Poll
		if len(opts.Media) > 0 {
			request.Media = &TweetMedia{
				MediaIDs: make([]string, len(opts.Media)),
			}
			for i, m := range opts.Media {
				request.Media.MediaIDs[i] = m.MediaKey
			}
		}
	}

	return request
}

// postTweetHelper handles the common tweet posting logic
// Rate limits:
// - 200 tweets per 15-minute window (user auth)
// - 50 tweet deletions per 15-minute window (user auth)
func (c *TwitterClient) postTweetHelper(ctx context.Context, endpoint string, request interface{}) (*Tweet, error) {
	// Log the request payload
	c.logger.WithField("request", request).Debug("sending tweet request")

	resp, err := c.makeRequest(ctx, "POST", endpoint, request)
	if err != nil {
		c.logger.WithError(err).Error("failed to post tweet")
		return nil, err
	}
	defer resp.Body.Close()

	// Read the entire response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Error("failed to read response body")
		return nil, err
	}

	// Log the raw response
	c.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"response":    string(bodyBytes),
		"endpoint":    endpoint,
	}).Debug("received twitter API response")

	// Recreate the response body for JSON decoding
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var tweetResponse TweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetResponse); err != nil {
		c.logger.WithError(err).WithField("body", string(bodyBytes)).Error("failed to decode tweet response")
		return nil, err
	}

	return tweetResponse.Data, nil
}
