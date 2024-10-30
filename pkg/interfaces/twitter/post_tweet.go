package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	// Create the request body structure
	requestBody := map[string]interface{}{
		"text": text,
	}

	// Add reply options if present
	if opts != nil && opts.ReplyOptions != nil {
		requestBody["reply"] = map[string]interface{}{
			"in_reply_to_tweet_id": opts.ReplyOptions.InReplyToTweetId,
		}
	}

	// Debug log the final request body
	requestJSON, _ := json.MarshalIndent(requestBody, "", "  ")
	logrus.WithFields(logrus.Fields{
		"endpoint":     c.config.TweetEndpoint,
		"request_body": string(requestJSON),
	}).Debug("sending tweet request to Twitter API")

	// Make the request
	resp, err := c.makeRequest(ctx, http.MethodPost, c.config.TweetEndpoint, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to post tweet: %w", err)
	}
	defer resp.Body.Close()

	var tweetResp TweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle potential errors
	if len(tweetResp.Errors) > 0 {
		return nil, fmt.Errorf("twitter API error: %s", tweetResp.Errors[0].Message)
	}

	// Unmarshal the single tweet response
	tweet, err := tweetResp.UnmarshalTweet()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tweet: %w", err)
	}

	// Add thread verification logging
	if resp.StatusCode == 201 {
		logrus.WithFields(logrus.Fields{
			"new_tweet_id":      tweet.ID,
			"conversation_id":   tweet.ConversationID,
			"referenced_tweets": tweet.ReferencedTweets,
		}).Debug("Thread verification")
	}

	return tweet, nil
}
