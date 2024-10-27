package twitter

import (
	"context"

	"github.com/sirupsen/logrus"
)

// TweetOptions is now just an alias for BaseOptions
type TweetOptions = BaseOptions

// CreateTweetRequest now embeds the base request
type CreateTweetRequest struct {
	BaseTweetRequest
}

func (c *TwitterClient) PostTweetAsync(ctx context.Context, text string, opts *TweetOptions) (chan *Tweet, chan error) {
	tweets := make(chan *Tweet, 1)
	errors := make(chan error, 1)

	go func() {
		defer close(tweets)
		defer close(errors)

		request := CreateTweetRequest{
			BaseTweetRequest: buildBaseRequest(text, opts),
		}

		logrus.WithFields(logrus.Fields{
			"text":     text,
			"endpoint": c.config.TweetEndpoint,
			"request":  request,
		}).Debug("sending tweet request to Twitter API")

		tweet, err := c.postTweetHelper(ctx, c.config.TweetEndpoint, request)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error":    err.Error(),
				"text":     text,
				"endpoint": c.config.TweetEndpoint,
				"request":  request,
			}).Error("failed to post tweet")
			errors <- err
			return
		}

		logrus.WithFields(logrus.Fields{
			"tweet_id":      tweet.ID,
			"text":          tweet.Text,
			"created_at":    tweet.CreatedAt,
			"author_id":     tweet.AuthorID,
			"full_response": tweet,
		}).Debug("complete Twitter API response")

		tweets <- tweet
	}()

	return tweets, errors
}

func (c *TwitterClient) PostTweet(ctx context.Context, text string, opts *TweetOptions) (*Tweet, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tweets, errs := c.PostTweetAsync(ctx, text, opts)

	select {
	case tweet := <-tweets:
		logrus.WithFields(logrus.Fields{
			"tweet_id":      tweet.ID,
			"text":          tweet.Text,
			"created_at":    tweet.CreatedAt,
			"author_id":     tweet.AuthorID,
			"full_response": tweet,
		}).Debug("tweet posted successfully")
		return tweet, nil
	case err := <-errs:
		return nil, err
	case <-ctx.Done():
		logrus.WithFields(logrus.Fields{
			"error": ctx.Err(),
			"text":  text,
		}).Debug("context cancelled while posting tweet")
		return nil, ctx.Err()
	}
}
