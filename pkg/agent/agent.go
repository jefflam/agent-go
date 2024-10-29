package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
)

const (
	defaultCheckMentionsInterval = 30 * time.Second
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
	if config.CheckMentions.Interval == 0 {
		config.CheckMentions.Interval = defaultCheckMentionsInterval
	}

	config.CheckMentions.ticker = time.NewTicker(config.CheckMentions.Interval)

	return &Agent{
		client:         config.TwitterClient,
		llm:            config.LLM,
		logger:         config.Logger,
		mentionsConfig: config.CheckMentions,
	}, nil
}

// Run starts the agent's mention checking loop
func (a *Agent) Run(ctx context.Context) error {
	log := a.logger.WithField("interval", a.mentionsConfig.Interval)
	log.Info("Starting mention monitoring")

	for {
		select {
		case <-ctx.Done():
			a.mentionsConfig.ticker.Stop()
			return ctx.Err()
		case <-a.mentionsConfig.ticker.C:
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
		return a.processMentions(ctx, resp.Tweet)
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
				"tweet_id":        tweet.ID,
				"author":          tweet.AuthorID,
				"text":            tweet.Text,
				"conversation_id": tweet.ConversationID,
			}).Info("Processing mention")

			// TODO: Add your tweet processing logic here
			// 1. Analyze tweet content using LLM
			// 2. Generate response
			// 3. Post reply
		}
	}
	return nil
}
