package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// TweetResponder handles responding to tweets that need replies
type TweetResponder struct {
	tweetStore *memory.TweetStore
	client     *twitter.TwitterClient
	logger     *logrus.Logger
	limiter    *rate.Limiter
}

// BatchProcessConfig holds configuration for batch processing
type BatchProcessConfig struct {
	BatchSize       int
	BatchDelay      time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	RateLimitWindow time.Duration
	TweetsPerWindow int
}

// DefaultBatchConfig returns default batch processing configuration
func DefaultBatchConfig() BatchProcessConfig {
	return BatchProcessConfig{
		BatchSize:       10,
		BatchDelay:      time.Minute,
		MaxRetries:      3,
		RetryDelay:      time.Minute,
		RateLimitWindow: 15 * time.Minute,
		TweetsPerWindow: 45,
	}
}

// NewTweetResponder creates a new TweetResponder instance
func NewTweetResponder(store *memory.TweetStore, client *twitter.TwitterClient, logger *logrus.Logger) *TweetResponder {
	// Twitter API v2 rate limit: 50 tweets per 15 minutes
	// Using a more conservative rate of 45 tweets per 15 minutes
	tweetsPerWindow := 45
	windowDuration := 15 * time.Minute
	r := rate.Every(windowDuration / time.Duration(tweetsPerWindow))

	return &TweetResponder{
		tweetStore: store,
		client:     client,
		logger:     logger,
		limiter:    rate.NewLimiter(r, 1), // burst size of 1 for conservative approach
	}
}

// ProcessTweetsNeedingReply finds and responds to tweets needing replies
func (tr *TweetResponder) ProcessTweetsNeedingReply(ctx context.Context) error {
	log := tr.logger.WithField("method", "ProcessTweetsNeedingReply")

	tweets, err := tr.tweetStore.RecallTweetsNeedingReply(ctx, tr.client)
	if err != nil {
		return fmt.Errorf("failed to recall tweets needing reply: %w", err)
	}

	log.WithField("tweets_count", len(tweets)).Info("Found tweets needing reply")

	// Create a channel for processing tweets with a buffer
	tweetChan := make(chan memory.TweetNeedingReply, len(tweets))
	for _, tweet := range tweets {
		tweetChan <- tweet
	}
	close(tweetChan)

	// Process tweets with rate limiting
	for tweet := range tweetChan {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Wait for rate limiter
			err := tr.limiter.Wait(ctx)
			if err != nil {
				log.WithError(err).Error("Rate limiter wait failed")
				continue
			}

			if err := tr.handleSingleReply(ctx, tweet); err != nil {
				log.WithError(err).WithFields(logrus.Fields{
					"tweet_id": tweet.TweetID,
					"error":    err,
				}).Error("Failed to handle reply")

				// Check if error is rate limit related
				if tr.isRateLimitError(err) {
					log.Info("Rate limit reached, pausing processing")
					time.Sleep(5 * time.Minute) // Add backoff when rate limit is hit
					continue
				}
			}

			// Add small delay between successful tweets for safety
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// ProcessTweetsInBatches processes tweets in controlled batches
func (tr *TweetResponder) ProcessTweetsInBatches(ctx context.Context, config BatchProcessConfig) error {
	log := tr.logger.WithField("method", "ProcessTweetsInBatches")

	tweets, err := tr.tweetStore.RecallTweetsNeedingReply(ctx, tr.client)
	if err != nil {
		return fmt.Errorf("failed to recall tweets needing reply: %w", err)
	}

	totalTweets := len(tweets)
	log.WithField("total_tweets", totalTweets).Info("Starting batch processing")

	for i := 0; i < totalTweets; i += config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			end := i + config.BatchSize
			if end > totalTweets {
				end = totalTweets
			}

			batch := tweets[i:end]
			log.WithFields(logrus.Fields{
				"batch_number": i/config.BatchSize + 1,
				"batch_size":   len(batch),
			}).Info("Processing batch")

			for _, tweet := range batch {
				if err := tr.processTweetWithRetry(ctx, tweet, config); err != nil {
					log.WithError(err).WithField("tweet_id", tweet.TweetID).Error("Failed to process tweet after retries")
				}
			}

			// Wait between batches
			if end < totalTweets {
				log.WithField("delay", config.BatchDelay).Info("Waiting between batches")
				time.Sleep(config.BatchDelay)
			}
		}
	}

	return nil
}

// processTweetWithRetry attempts to process a tweet with retries
func (tr *TweetResponder) processTweetWithRetry(ctx context.Context, tweet memory.TweetNeedingReply, config BatchProcessConfig) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		err := tr.limiter.Wait(ctx)
		if err != nil {
			return fmt.Errorf("rate limiter wait failed: %w", err)
		}

		if err := tr.handleSingleReply(ctx, tweet); err != nil {
			lastErr = err
			if tr.isRateLimitError(err) {
				tr.logger.WithField("retry_delay", config.RetryDelay).Info("Rate limit hit, waiting before retry")
				time.Sleep(config.RetryDelay)
				continue
			}
		} else {
			return nil
		}

		time.Sleep(config.RetryDelay)
	}

	return fmt.Errorf("failed after %d retries: %w", config.MaxRetries, lastErr)
}

// handleSingleReply processes a single tweet that needs a reply
func (tr *TweetResponder) handleSingleReply(ctx context.Context, tweet memory.TweetNeedingReply) error {
	log := tr.logger.WithFields(logrus.Fields{
		"method":          "handleSingleReply",
		"tweet_id":        tweet.TweetID,
		"conversation_id": tweet.ConversationID,
	})

	// TODO: Generate reply text using your AI/LLM implementation
	replyText := "This is a placeholder reply" // Replace with actual AI-generated reply

	// Post the reply
	params := twitter.PostReplyThreadParams{
		Text:           replyText,
		ReplyToID:      tweet.TweetID,
		ConversationID: tweet.ConversationID,
	}

	postedTweet, err := tr.client.PostReplyThread(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to post reply: %w", err)
	}

	log.WithFields(logrus.Fields{
		"reply_tweet_id": postedTweet.ID,
		"original_tweet": tweet.TweetID,
	}).Info("Successfully posted reply")

	return nil
}

// isRateLimitError checks if the error is related to rate limiting
func (tr *TweetResponder) isRateLimitError(err error) bool {
	// Implement based on your Twitter client's error types
	// Example: return strings.Contains(err.Error(), "rate limit exceeded")
	return false
}
