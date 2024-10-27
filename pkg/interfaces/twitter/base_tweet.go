package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/sirupsen/logrus"
)

// TweetMedia represents media attachments in a tweet
type TweetMedia struct {
	MediaIDs []string `json:"media_ids"`
}

// BaseTweetRequest contains common fields for all tweet-related requests
type BaseTweetRequest struct {
	Text                  string      `json:"text"`
	ReplySettings         string      `json:"reply_settings,omitempty"`
	ForSuperFollowersOnly bool        `json:"for_super_followers_only,omitempty"`
	QuoteTweetID          string      `json:"quote_tweet_id,omitempty"`
	ReplyTo               string      `json:"in_reply_to_tweet_id,omitempty"`
	Poll                  *Poll       `json:"poll,omitempty"`
	Media                 *TweetMedia `json:"media,omitempty"`
}

// BaseOptions contains common optional parameters for tweet operations
type BaseOptions struct {
	ReplyTo               string  `json:"reply_to,omitempty"`
	QuoteTweetID          string  `json:"quote_tweet_id,omitempty"`
	Poll                  *Poll   `json:"poll,omitempty"`
	Media                 []Media `json:"media,omitempty"`
	ReplySettings         string  `json:"reply_settings,omitempty"`
	ForSuperFollowersOnly bool    `json:"for_super_followers_only,omitempty"`
}

// buildBaseRequest creates a BaseTweetRequest from text and options
func buildBaseRequest(text string, opts *BaseOptions) BaseTweetRequest {
	request := BaseTweetRequest{
		Text: text,
	}

	if opts != nil {
		request.ReplySettings = opts.ReplySettings
		request.ForSuperFollowersOnly = opts.ForSuperFollowersOnly
		request.QuoteTweetID = opts.QuoteTweetID
		request.ReplyTo = opts.ReplyTo
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
