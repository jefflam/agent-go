package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/lisanmuaddib/agent-go/pkg/thoughts"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// TweetResponder handles responding to tweets that need replies
type TweetResponder struct {
	tweetStore     *memory.TweetStore
	client         *twitter.TwitterClient
	logger         *logrus.Logger
	limiter        *rate.Limiter
	replyGenerator thoughts.MentionReplyGenerator
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
func NewTweetResponder(
	store *memory.TweetStore,
	client *twitter.TwitterClient,
	logger *logrus.Logger,
	replyGenerator thoughts.MentionReplyGenerator,
) *TweetResponder {
	// Twitter API v2 rate limit: 50 tweets per 15 minutes
	// Using a more conservative rate of 45 tweets per 15 minutes
	tweetsPerWindow := 45
	windowDuration := 15 * time.Minute
	r := rate.Every(windowDuration / time.Duration(tweetsPerWindow))

	return &TweetResponder{
		tweetStore:     store,
		client:         client,
		logger:         logger,
		limiter:        rate.NewLimiter(r, 1), // burst size of 1 for conservative approach
		replyGenerator: replyGenerator,
	}
}

// ProcessTweetsNeedingReply finds and responds to tweets needing replies
func (tr *TweetResponder) ProcessTweetsNeedingReply(ctx context.Context) error {
	log := tr.logger.WithField("method", "ProcessTweetsNeedingReply")

	threads, err := tr.tweetStore.RecallTweetsNeedingReply(ctx, tr.client)
	if err != nil {
		return fmt.Errorf("failed to recall tweets needing reply: %w", err)
	}

	log.WithField("threads_count", len(threads)).Info("Found conversation threads needing reply")

	// Create a channel for processing conversation threads with a buffer
	threadChan := make(chan memory.ConversationThread, len(threads))
	for _, thread := range threads {
		threadChan <- thread
	}
	close(threadChan)

	// Process threads with rate limiting
	for thread := range threadChan {
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

			if err := tr.handleSingleReply(ctx, thread); err != nil {
				log.WithError(err).WithFields(logrus.Fields{
					"conversation_id": thread.ConversationID,
					"tweets_count":    len(thread.Tweets),
					"error":           err,
				}).Error("Failed to handle reply")

				if tr.isRateLimitError(err) {
					log.Info("Rate limit reached, pausing processing")
					time.Sleep(5 * time.Minute)
					continue
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// handleSingleReply processes a single tweet that needs a reply
func (tr *TweetResponder) handleSingleReply(ctx context.Context, thread memory.ConversationThread) error {
	log := tr.logger.WithFields(logrus.Fields{
		"method":          "handleSingleReply",
		"conversation_id": thread.ConversationID,
		"tweets_count":    len(thread.Tweets),
	})

	// Get the most recent tweet to reply to
	lastTweet := thread.Tweets[len(thread.Tweets)-1]

	// Build conversation context in a format suitable for LLM processing
	var conversationContext strings.Builder
	conversationContext.WriteString("Previous conversation:\n")
	for i, tweet := range thread.Tweets {
		// Skip the last tweet as it will be the main tweet we're responding to
		if i == len(thread.Tweets)-1 {
			continue
		}
		// Format: "@username (name): message"
		conversationContext.WriteString(fmt.Sprintf("@%s (%s): %s\n",
			tweet.AuthorUsername,
			tweet.AuthorName,
			tweet.Text,
		))
	}
	conversationContext.WriteString("\nCurrent tweet to reply to:\n")
	conversationContext.WriteString(fmt.Sprintf("@%s (%s): %s",
		lastTweet.AuthorUsername,
		lastTweet.AuthorName,
		lastTweet.Text,
	))

	// Generate AI reply using the mention reply generator
	config := thoughts.MentionReplyConfig{
		TweetText:           lastTweet.Text,               // The tweet we're directly replying to
		ConversationContext: conversationContext.String(), // Full conversation history
		MaxLength:           280,                          // Twitter's character limit
		Temperature:         0.7,                          // Adjust as needed
		AuthorUsername:      lastTweet.AuthorUsername,     // Who we're replying to
		AuthorName:          lastTweet.AuthorName,         // Their display name
		Category:            lastTweet.Category,           // Type of interaction
		Language:            lastTweet.Lang,               // Tweet language
	}

	replyText, err := tr.replyGenerator.GenerateReply(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to generate reply: %w", err)
	}

	// Post the reply using existing PostReplyThread implementation
	params := twitter.PostReplyThreadParams{
		Text:           replyText,
		ReplyToID:      lastTweet.TweetID,
		ConversationID: thread.ConversationID,
	}

	postedTweet, err := tr.client.PostReplyThread(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to post reply: %w", err)
	}

	log.WithFields(logrus.Fields{
		"reply_tweet_id": postedTweet.ID,
		"reply_to_tweet": lastTweet.TweetID,
		"reply_text":     replyText,
		"context_length": len(thread.Tweets),
	}).Info("Successfully posted reply")

	return nil
}

// ProcessTweetsInBatches processes tweets in controlled batches
func (tr *TweetResponder) ProcessTweetsInBatches(ctx context.Context, config BatchProcessConfig) error {
	log := tr.logger.WithField("method", "ProcessTweetsInBatches")

	threads, err := tr.tweetStore.RecallTweetsNeedingReply(ctx, tr.client)
	if err != nil {
		return fmt.Errorf("failed to recall tweets needing reply: %w", err)
	}

	totalThreads := len(threads)
	log.WithField("total_threads", totalThreads).Info("Starting batch processing")

	for i := 0; i < totalThreads; i += config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			end := i + config.BatchSize
			if end > totalThreads {
				end = totalThreads
			}

			batch := threads[i:end]
			log.WithFields(logrus.Fields{
				"batch_number": i/config.BatchSize + 1,
				"batch_size":   len(batch),
			}).Info("Processing batch")

			for _, thread := range batch {
				if err := tr.handleSingleReply(ctx, thread); err != nil {
					log.WithError(err).WithField("conversation_id", thread.ConversationID).
						Error("Failed to process thread")
				}
			}

			if end < totalThreads {
				log.WithField("delay", config.BatchDelay).Info("Waiting between batches")
				time.Sleep(config.BatchDelay)
			}
		}
	}

	return nil
}

// isRateLimitError checks if the error is related to rate limiting
func (tr *TweetResponder) isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for Twitter API v2 rate limit errors
	errStr := err.Error()
	rateLimitKeywords := []string{
		"rate limit exceeded",
		"too many requests",
		"429 Too Many Requests",
		"rate_limit_exceeded",
		"x-rate-limit-remaining: 0",
		"x-rate-limit-reset",
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(strings.ToLower(errStr), keyword) {
			tr.logger.WithFields(logrus.Fields{
				"error":    errStr,
				"type":     "rate_limit",
				"endpoint": "tweets",
				"window":   "15m",
				"limit":    "50 tweets",
			}).Debug("Twitter API v2 rate limit error detected")
			return true
		}
	}

	return false
}
