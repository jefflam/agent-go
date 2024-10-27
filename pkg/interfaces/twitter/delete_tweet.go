package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

// DeleteTweetResponse represents the response from deleting a tweet
type DeleteTweetResponse struct {
	Data struct {
		Deleted bool `json:"deleted"`
	} `json:"data"`
}

// deleteTweetHelper handles the common tweet deletion logic
func (c *TwitterClient) deleteTweetHelper(ctx context.Context, tweetID string) (bool, error) {
	c.logger.WithFields(logrus.Fields{
		"tweet_id": tweetID,
		"method":   "DELETE",
	}).Debug("attempting to delete tweet")

	endpoint := fmt.Sprintf("%s/%s", c.config.TweetEndpoint, tweetID)

	// Log before making request
	c.logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"method":   "DELETE",
	}).Debug("making delete request to Twitter API")

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"tweet_id": tweetID,
			"endpoint": endpoint,
		}).Error("failed to delete tweet")
		return false, err
	}
	defer resp.Body.Close()

	// Read the full response body for logging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"tweet_id": tweetID,
		}).Error("failed to read response body")
		return false, err
	}

	// Enhanced debug logging for response
	c.logger.WithFields(logrus.Fields{
		"endpoint":     endpoint,
		"response":     string(body),
		"status_code":  resp.StatusCode,
		"headers":      resp.Header,
		"tweet_id":     tweetID,
		"content_type": resp.Header.Get("Content-Type"),
	}).Debug("received delete tweet response")

	// Create a new reader with the body for JSON decoding
	var deleteResponse DeleteTweetResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&deleteResponse); err != nil {
		c.logger.WithError(err).Error("failed to decode delete response")
		return false, err
	}

	return deleteResponse.Data.Deleted, nil
}

// DeleteTweetAsync deletes a tweet asynchronously
func (c *TwitterClient) DeleteTweetAsync(ctx context.Context, tweetID string) (chan bool, chan error) {
	c.logger.WithFields(logrus.Fields{
		"tweet_id": tweetID,
	}).Debug("starting async tweet deletion")

	deleted := make(chan bool, 1)
	errors := make(chan error, 1)

	go func() {
		defer close(deleted)
		defer close(errors)

		c.logger.WithFields(logrus.Fields{
			"tweet_id": tweetID,
		}).Debug("executing async deletion")

		isDeleted, err := c.deleteTweetHelper(ctx, tweetID)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"tweet_id": tweetID,
			}).Error("async deletion failed")
			errors <- err
			return
		}

		c.logger.WithFields(logrus.Fields{
			"tweet_id":   tweetID,
			"is_deleted": isDeleted,
		}).Debug("async deletion completed")
		deleted <- isDeleted
	}()

	return deleted, errors
}

// DeleteTweet deletes a tweet synchronously
func (c *TwitterClient) DeleteTweet(ctx context.Context, tweetID string) (bool, error) {
	c.logger.WithFields(logrus.Fields{
		"tweet_id": tweetID,
	}).Debug("starting synchronous tweet deletion")

	if ctx == nil {
		ctx = context.Background()
		c.logger.Debug("created background context for null context")
	}

	deleted, errs := c.DeleteTweetAsync(ctx, tweetID)

	select {
	case isDeleted := <-deleted:
		c.logger.WithFields(logrus.Fields{
			"tweet_id":   tweetID,
			"is_deleted": isDeleted,
		}).Debug("synchronous deletion completed")
		return isDeleted, nil
	case err := <-errs:
		c.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"tweet_id": tweetID,
		}).Error("synchronous deletion failed")
		return false, err
	case <-ctx.Done():
		c.logger.WithFields(logrus.Fields{
			"tweet_id": tweetID,
			"error":    ctx.Err().Error(),
		}).Error("deletion context cancelled")
		return false, ctx.Err()
	}
}
