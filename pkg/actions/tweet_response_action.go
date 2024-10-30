package actions

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// TweetResponseOptions configures the tweet response action
type TweetResponseOptions struct {
	Interval    time.Duration
	BatchConfig BatchProcessConfig
}

// TweetResponseAction implements the Action interface for responding to tweets
type TweetResponseAction struct {
	responder *TweetResponder
	logger    *logrus.Logger
	options   TweetResponseOptions
}

// NewTweetResponseAction creates a new tweet response action
func NewTweetResponseAction(
	responder *TweetResponder,
	logger *logrus.Logger,
	options TweetResponseOptions,
) *TweetResponseAction {
	return &TweetResponseAction{
		responder: responder,
		logger:    logger,
		options:   options,
	}
}

// Name implements the Action interface
func (t *TweetResponseAction) Name() string {
	return "tweet_response"
}

// Execute implements the Action interface
func (t *TweetResponseAction) Execute(ctx context.Context) error {
	log := t.logger.WithField("action", t.Name())

	ticker := time.NewTicker(t.options.Interval)
	defer ticker.Stop()

	log.Info("Starting tweet response action")

	for {
		select {
		case <-ctx.Done():
			log.Info("Tweet response action stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := t.responder.ProcessTweetsInBatches(ctx, t.options.BatchConfig); err != nil {
				log.WithError(err).Error("Failed to process tweets needing reply")
				// Continue running even if we encounter an error
				continue
			}
		}
	}
}

// Stop implements the Action interface
func (t *TweetResponseAction) Stop() {
	log := t.logger.WithField("action", t.Name())
	log.Info("Stopping tweet response action")
}
