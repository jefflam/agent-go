package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
)

const (
	defaultInterval = 30 * time.Second
)

// New creates a new Agent instance
func New(config Config) (*Agent, error) {
	if config.LLM == nil {
		return nil, fmt.Errorf("LLM is required")
	}
	if config.TwitterClient == nil {
		return nil, fmt.Errorf("TwitterClient is required")
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}
	if config.Interval == 0 {
		config.Interval = defaultInterval
	}

	return &Agent{
		client:   config.TwitterClient,
		llm:      config.LLM,
		logger:   config.Logger,
		interval: config.Interval,
		ticker:   time.NewTicker(config.Interval),
	}, nil
}

// Run starts the agent's mention checking loop
func (a *Agent) Run(ctx context.Context) error {
	log := a.logger.WithField("interval", a.interval)
	log.Info("Starting mention monitoring")

	for {
		select {
		case <-ctx.Done():
			a.ticker.Stop()
			return ctx.Err()
		case <-a.ticker.C:
			if err := a.checkMentions(ctx); err != nil {
				log.WithError(err).Error("Failed to check mentions")
				// Continue running despite errors
			}
		}
	}
}

func (a *Agent) checkMentions(ctx context.Context) error {
	log := a.logger.WithField("method", "checkMentions")
	log.Debug("Checking for new mentions")

	params := twitter.GetUserMentionsParams{
		MaxResults: 100,
	}

	dataChan, errChan := a.client.GetUserMentions(ctx, params)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case resp, ok := <-dataChan:
		if !ok {
			return nil
		}
		return a.processMentions(ctx, resp)
	}
}

func (a *Agent) processMentions(ctx context.Context, resp *twitter.TweetResponse) error {
	if resp == nil || len(resp.Data) == 0 {
		return nil
	}

	for _, tweet := range resp.Data {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			a.logger.WithFields(logrus.Fields{
				"tweet_id": tweet.ID,
				"author":   tweet.AuthorID,
				"text":     tweet.Text,
			}).Info("Processing mention")

			// TODO: Add your tweet processing logic here
			// 1. Analyze tweet content using LLM
			// 2. Generate response
			// 3. Post reply
		}
	}
	return nil
}
