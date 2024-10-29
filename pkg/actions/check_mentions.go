package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

type MentionsHandler struct {
	client     *twitter.TwitterClient
	llm        llms.Model
	logger     *logrus.Logger
	ticker     *time.Ticker
	options    MentionsOptions
	done       chan struct{}
	tweetStore *memory.TweetStore
}

type MentionsOptions struct {
	Interval   time.Duration
	MaxResults int
}

func NewMentionsHandler(client *twitter.TwitterClient, llm llms.Model, logger *logrus.Logger, options MentionsOptions) (*MentionsHandler, error) {
	if options.Interval == 0 {
		options.Interval = 30 * time.Second
	}
	if options.MaxResults == 0 {
		options.MaxResults = 100
	}

	// Initialize tweet store
	tweetStore, err := memory.NewTweetStore(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create tweet store: %w", err)
	}

	return &MentionsHandler{
		client:     client,
		llm:        llm,
		logger:     logger,
		ticker:     time.NewTicker(options.Interval),
		options:    options,
		done:       make(chan struct{}),
		tweetStore: tweetStore,
	}, nil
}

// Name returns the unique identifier for this action
func (h *MentionsHandler) Name() string {
	return "mentions_handler"
}

// Execute implements the Action interface
func (h *MentionsHandler) Execute(ctx context.Context) error {
	return h.Start(ctx)
}

// Stop implements the Action interface
func (h *MentionsHandler) Stop() {
	h.ticker.Stop()
	close(h.done)
}

func (h *MentionsHandler) Start(ctx context.Context) error {
	log := h.logger.WithField("interval", h.options.Interval)
	log.Info("Starting mention monitoring")

	for {
		select {
		case <-ctx.Done():
			h.ticker.Stop()
			return ctx.Err()
		case <-h.done:
			return nil
		case <-h.ticker.C:
			if err := h.CheckMentions(ctx); err != nil {
				log.WithError(err).Error("Failed to check mentions")
			}
		}
	}
}

func (h *MentionsHandler) CheckMentions(ctx context.Context) error {
	log := h.logger.WithField("method", "CheckMentions")
	log.Debug("Checking for new mentions")

	params := twitter.GetUserMentionsParams{
		MaxResults: h.options.MaxResults,
		TweetFields: []string{
			"id",
			"text",
			"author_id",
			"conversation_id",
			"created_at",
			"entities",
			"geo",
			"in_reply_to_user_id",
			"lang",
			"public_metrics",
			"referenced_tweets",
			"reply_settings",
			"source",
		},
		Expansions: []string{
			"author_id",
			"referenced_tweets.id",
			"in_reply_to_user_id",
			"entities.mentions.username",
			"referenced_tweets.id.author_id",
		},
	}

	dataChan, errChan := h.client.GetUserMentions(ctx, params)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case resp, ok := <-dataChan:
		if !ok {
			return nil
		}
		return h.processMentions(ctx, resp)
	}
}

func (h *MentionsHandler) processMentions(ctx context.Context, resp *twitter.MentionResponse) error {
	if resp == nil {
		return nil
	}

	tweets, err := resp.UnmarshalTweets()
	if err != nil {
		return fmt.Errorf("failed to unmarshal tweets: %w", err)
	}

	for _, tweet := range tweets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log := h.logger.WithFields(logrus.Fields{
				"tweet_id":        tweet.ID,
				"author_id":       tweet.AuthorID,
				"text":            tweet.Text,
				"conversation_id": tweet.ConversationID,
				"created_at":      tweet.CreatedAt,
				"reply_settings":  tweet.ReplySettings,
			})

			// Determine the category of the tweet
			category := memory.DetermineTweetCategory(tweet)

			// Store the tweet with all its metadata
			if err := h.tweetStore.SaveTweet(tweet, category); err != nil {
				log.WithError(err).Error("Failed to save tweet")
				continue
			}

			log.Info("Processed mention")
		}
	}

	return nil
}
