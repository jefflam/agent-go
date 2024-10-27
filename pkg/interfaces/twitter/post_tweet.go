package twitter

import (
	"context"
	"encoding/json"
)

// TweetOptions represents optional parameters for creating a tweet
type TweetOptions struct {
	ReplyTo               string  `json:"reply_to,omitempty"`
	QuoteTweetID          string  `json:"quote_tweet_id,omitempty"`
	Poll                  *Poll   `json:"poll,omitempty"`
	Media                 []Media `json:"media,omitempty"`
	ReplySettings         string  `json:"reply_settings,omitempty"`
	ForSuperFollowersOnly bool    `json:"for_super_followers_only,omitempty"`
}

// CreateTweetRequest represents the request body for creating a tweet
type CreateTweetRequest struct {
	Text                  string      `json:"text"`
	ReplySettings         string      `json:"reply_settings,omitempty"`
	ForSuperFollowersOnly bool        `json:"for_super_followers_only,omitempty"`
	QuoteTweetID          string      `json:"quote_tweet_id,omitempty"`
	ReplyTo               string      `json:"in_reply_to_tweet_id,omitempty"`
	Poll                  *Poll       `json:"poll,omitempty"`
	Media                 *TweetMedia `json:"media,omitempty"`
}

type TweetMedia struct {
	MediaIDs []string `json:"media_ids,omitempty"`
}

// PostTweetAsync creates a new tweet asynchronously and returns channels for the response and errors
func (c *TwitterClient) PostTweetAsync(ctx context.Context, text string, opts *TweetOptions) (chan *Tweet, chan error) {
	tweets := make(chan *Tweet, 1)
	errors := make(chan error, 1)

	go func() {
		defer close(tweets)
		defer close(errors)

		endpoint := c.config.TweetEndpoint
		request := CreateTweetRequest{
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

		resp, err := c.makeRequest(ctx, "POST", endpoint, request)
		if err != nil {
			c.logger.WithError(err).Error("failed to post tweet")
			errors <- err
			return
		}
		defer resp.Body.Close()

		var tweetResponse TweetResponse
		if err := json.NewDecoder(resp.Body).Decode(&tweetResponse); err != nil {
			c.logger.WithError(err).Error("failed to decode tweet response")
			errors <- err
			return
		}

		tweets <- tweetResponse.Data
	}()

	return tweets, errors
}

// PostTweet creates a new tweet synchronously
func (c *TwitterClient) PostTweet(ctx context.Context, text string, opts *TweetOptions) (*Tweet, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tweets, errs := c.PostTweetAsync(ctx, text, opts)

	// Wait for either a response or an error
	select {
	case tweet := <-tweets:
		return tweet, nil
	case err := <-errs:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PostReply creates a reply to an existing tweet
func (c *TwitterClient) PostReply(ctx context.Context, text, replyToID string) (*Tweet, error) {
	return c.PostTweet(ctx, text, &TweetOptions{
		ReplyTo: replyToID,
	})
}

// PostQuote creates a quote tweet
func (c *TwitterClient) PostQuote(ctx context.Context, text, quoteTweetID string) (*Tweet, error) {
	return c.PostTweet(ctx, text, &TweetOptions{
		QuoteTweetID: quoteTweetID,
	})
}
