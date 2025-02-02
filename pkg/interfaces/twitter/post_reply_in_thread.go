package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
)

// PostReplyThreadParams holds the parameters for posting a reply in a thread
type PostReplyThreadParams struct {
	Text           string
	ReplyToID      string
	ConversationID string
}

// PostReplyThread creates a reply that maintains the conversation thread
func (c *TwitterClient) PostReplyThread(ctx context.Context, params PostReplyThreadParams) (*Tweet, error) {
	log := c.logger.WithFields(logrus.Fields{
		"method":          "PostReplyThread",
		"text":            params.Text,
		"reply_to_id":     params.ReplyToID,
		"conversation_id": params.ConversationID,
	})

	// If no conversation ID is provided, fetch it from the tweet we're replying to
	if params.ConversationID == "" {
		// Get the original tweet to fetch its conversation_id
		tweetChan, errChan := c.GetTweetByID(ctx, GetTweetByIDParams{
			TweetID: params.ReplyToID,
		})

		select {
		case resp := <-tweetChan:
			if tweet, err := resp.UnmarshalTweet(); err == nil && tweet != nil {
				params.ConversationID = tweet.ConversationID
			}
		case err := <-errChan:
			log.WithError(err).Error("failed to fetch original tweet")
			return nil, fmt.Errorf("failed to fetch original tweet: %w", err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Add validation before creating request
	if params.ReplyToID == "" {
		return nil, fmt.Errorf("reply_to_id is required for posting a reply")
	}

	// Validate tweet ID format (1-19 digits)
	if !regexp.MustCompile(`^[0-9]{1,19}$`).MatchString(params.ReplyToID) {
		return nil, fmt.Errorf("invalid reply_to_id format: must be 1-19 digits")
	}

	// Create the reply with thread information
	opts := &BaseOptions{
		ReplyOptions: &ReplyOptions{
			InReplyToTweetId: params.ReplyToID,
		},
	}

	// Log the request body before sending
	requestBody := map[string]interface{}{
		"text": params.Text,
		"reply": map[string]interface{}{
			"in_reply_to_tweet_id": params.ReplyToID,
		},
	}

	requestJSON, err := json.MarshalIndent(requestBody, "", "  ")
	if err != nil {
		log.WithError(err).Error("failed to marshal request body for logging")
	} else {
		log.WithFields(logrus.Fields{
			"request_body": string(requestJSON),
			"reply_to_id":  params.ReplyToID,
		}).Debug("Sending tweet reply request")
	}

	tweet, err := c.PostTweet(ctx, params.Text, opts)
	if err != nil {
		log.WithFields(logrus.Fields{
			"request_body": string(requestJSON),
		}).WithError(err).Error("failed to post reply tweet in thread")
		return nil, fmt.Errorf("failed to post reply tweet in thread: %w", err)
	}

	log.WithFields(logrus.Fields{
		"tweet_id":        tweet.ID,
		"conversation_id": tweet.ConversationID,
	}).Debug("successfully posted reply tweet in thread")

	return tweet, nil
}
