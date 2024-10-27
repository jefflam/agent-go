package twitter

import (
	"context"

	"github.com/sirupsen/logrus"
)

// PostReply creates a reply to an existing tweet
func (c *TwitterClient) PostReply(ctx context.Context, text, replyToID string) (*Tweet, error) {
	logrus.WithFields(logrus.Fields{
		"text":      text,
		"replyToID": replyToID,
	}).Debug("attempting to post reply tweet")

	opts := &BaseOptions{
		ReplyTo: replyToID,
	}

	tweet, err := c.PostTweet(ctx, text, opts)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"text":      text,
			"replyToID": replyToID,
		}).Error("failed to post reply tweet")
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"tweetID":   tweet.ID,
		"text":      text,
		"replyToID": replyToID,
		"response":  tweet,
	}).Debug("successfully posted reply tweet")

	return tweet, nil
}
