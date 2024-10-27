package twitter

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// PostQuote creates a quote tweet
func (c *TwitterClient) PostQuote(ctx context.Context, text, quoteTweetID string) (*Tweet, error) {
	log := c.log.WithFields(logrus.Fields{
		"method":       "PostQuote",
		"text":         text,
		"quoteTweetID": quoteTweetID,
	})

	log.Debug("attempting to post quote tweet")

	opts := &BaseOptions{
		QuoteTweetID: quoteTweetID,
	}

	tweet, err := c.PostTweet(ctx, text, opts)
	if err != nil {
		log.WithError(err).Error("failed to post quote tweet")
		return nil, err
	}

	// Enhanced logging of the response
	log.WithFields(logrus.Fields{
		"tweet_id":          tweet.ID,
		"tweet_text":        tweet.Text,
		"created_at":        tweet.CreatedAt,
		"complete_response": fmt.Sprintf("%+v", tweet),
	}).Debug("twitter API response details")

	return tweet, nil
}
