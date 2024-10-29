package actions

import (
	"context"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

type MentionsHandler struct {
	client  *twitter.TwitterClient
	llm     llms.Model
	logger  *logrus.Logger
	ticker  *time.Ticker
	options MentionsOptions
	done    chan struct{}
}

type MentionsOptions struct {
	Interval   time.Duration
	MaxResults int
}

func NewMentionsHandler(client *twitter.TwitterClient, llm llms.Model, logger *logrus.Logger, options MentionsOptions) *MentionsHandler {
	if options.Interval == 0 {
		options.Interval = 30 * time.Second
	}
	if options.MaxResults == 0 {
		options.MaxResults = 100
	}

	return &MentionsHandler{
		client:  client,
		llm:     llm,
		logger:  logger,
		ticker:  time.NewTicker(options.Interval),
		options: options,
		done:    make(chan struct{}),
	}
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
		return h.processMentions(ctx, resp.Tweet)
	}
}

func (h *MentionsHandler) processMentions(ctx context.Context, resp *twitter.TweetResponse) error {
	if resp == nil {
		return nil
	}

	tweets, err := resp.UnmarshalTweets()
	if err != nil {
		return err
	}

	for _, tweet := range tweets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			h.logger.WithFields(logrus.Fields{
				"tweet_id":        tweet.ID,
				"author":          tweet.AuthorID,
				"text":            tweet.Text,
				"conversation_id": tweet.ConversationID,
			}).Info("Processing mention")

			// TODO: Implement tweet processing logic
			// This will be implemented in a separate PR
		}
	}
	return nil
}
