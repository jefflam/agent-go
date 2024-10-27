package agent

import (
	"context"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"

	"github.com/sirupsen/logrus"
)

// MentionProcessor handles the processing of Twitter mentions
type MentionProcessor struct {
	client *twitter.TwitterClient
	logger *logrus.Logger
}

// NewMentionProcessor creates a new MentionProcessor instance
func NewMentionProcessor(client *twitter.TwitterClient, logger *logrus.Logger) *MentionProcessor {
	if logger == nil {
		logger = logrus.New()
	}

	return &MentionProcessor{
		client: client,
		logger: logger,
	}
}

// ProcessMentions handles a single batch of mention processing
func (mp *MentionProcessor) ProcessMentions(ctx context.Context) error {
	log := mp.logger.WithField("method", "ProcessMentions")
	log.Debug("Processing mentions")

	params := twitter.GetUserMentionsParams{
		MaxResults: 100,
	}

	dataChan, errChan := mp.client.GetUserMentions(ctx, params)

	// Process just one batch of mentions
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	case resp, ok := <-dataChan:
		if !ok {
			return nil // Channel closed, no data
		}
		mp.processTweetResponse(ctx, resp)
	}

	return nil
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

				// TODO: Implement mention processing logic
				// 1. Analyze the tweet content
				// 2. Generate appropriate responses
				// 3. Post replies
			}
		}
	}
}
