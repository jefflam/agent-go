package agent

import (
	"context"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// MentionProcessor implements the Task interface for handling Twitter mentions
type MentionProcessor struct {
	client  *twitter.TwitterClient
	llm     llms.Model
	logger  *logrus.Logger
	ticker  *time.Ticker
	stopped chan struct{}
}

// NewMentionProcessor creates a new MentionProcessor instance
func NewMentionProcessor(client *twitter.TwitterClient, llm llms.Model, logger *logrus.Logger, interval time.Duration) *MentionProcessor {
	if logger == nil {
		logger = logrus.New()
	}

	return &MentionProcessor{
		client:  client,
		llm:     llm,
		logger:  logger,
		ticker:  time.NewTicker(interval),
		stopped: make(chan struct{}),
	}
}

// Run implements the Task interface
func (mp *MentionProcessor) Run(ctx context.Context) error {
	log := mp.logger.WithField("task", "mentions")
	log.Info("Starting mentions processor")

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping mentions processor")
			return ctx.Err()
		case <-mp.stopped:
			log.Info("Mentions processor stopped")
			return nil
		case <-mp.ticker.C:
			if err := mp.ProcessMentions(ctx); err != nil {
				log.WithError(err).Error("Failed to process mentions")
				// Continue running despite errors
			}
		}
	}
}

// Stop implements the Task interface
func (mp *MentionProcessor) Stop() {
	mp.ticker.Stop()
	close(mp.stopped)
}

// Type implements the Task interface
func (mp *MentionProcessor) Type() TaskType {
	return TaskMentions
}

// ProcessMentions handles mention processing
func (mp *MentionProcessor) ProcessMentions(ctx context.Context) error {
	log := mp.logger.WithField("method", "ProcessMentions")
	log.Debug("Processing mentions")

	params := twitter.GetUserMentionsParams{
		MaxResults: 100,
	}

	dataChan, errChan := mp.client.GetUserMentions(ctx, params)

	// Process all available mentions
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			if err != nil {
				return err
			}
			return nil // Error channel closed normally
		case resp, ok := <-dataChan:
			if !ok {
				return nil // Data channel closed, no more mentions
			}
			if resp.Tweet != nil {
				mp.processTweetResponse(ctx, resp.Tweet)
			}
		}
	}
}

func (mp *MentionProcessor) processTweetResponse(ctx context.Context, resp *twitter.TweetResponse) {
	log := mp.logger.WithField("method", "processTweetResponse")

	select {
	case <-ctx.Done():
		log.Info("Context cancelled, stopping mention processing")
		return
	default:
		if resp == nil || len(resp.Data) == 0 {
			log.Debug("No mentions to process")
			return
		}

		for _, tweet := range resp.Data {
			select {
			case <-ctx.Done():
				log.Info("Context cancelled during tweet processing")
				return
			default:
				log.WithFields(logrus.Fields{
					"tweet_id":        tweet.ID,
					"author":          tweet.AuthorID,
					"text":            tweet.Text,
					"conversation_id": tweet.ConversationID,
					"created_at":      tweet.CreatedAt,
					"lang":            tweet.Lang,
				}).Info("Processing mention")

				// Process tweet entities
				mp.processEntities(tweet)

				// TODO: Implement mention processing logic using LLM
				// 1. Generate response using mp.llm
				// 2. Post reply using mp.client
			}
		}
	}
}

func (mp *MentionProcessor) processEntities(tweet twitter.Tweet) {
	log := mp.logger.WithField("tweet_id", tweet.ID)

	if tweet.Entities.Mentions != nil {
		for _, mention := range tweet.Entities.Mentions {
			log.WithFields(logrus.Fields{
				"username": mention.Username,
				"user_id":  mention.ID,
			}).Debug("Found mention in tweet")
		}
	}

	if len(tweet.ReferencedTweets) > 0 {
		for _, ref := range tweet.ReferencedTweets {
			log.WithFields(logrus.Fields{
				"type":          ref.Type,
				"referenced_id": ref.ID,
			}).Debug("Found referenced tweet")
		}
	}
}
