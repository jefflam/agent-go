package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/internal/personality/traits"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/thoughts"
	"github.com/sirupsen/logrus"
)

const (
	// MaxTweetLength is Twitter's maximum allowed characters
	MaxTweetLength = 280
)

// OriginalThoughtConfig holds configuration for posting thoughts to Twitter
type OriginalThoughtConfig struct {
	Topic       string
	Temperature float64 // Controls randomness of thought generation
}

// OriginalThoughtPoster handles posting thoughts to Twitter
type OriginalThoughtPoster struct {
	thoughtGen thoughts.OriginalThoughtGenerator
	twitter    *twitter.TwitterClient
}

// NewOriginalThoughtPoster creates a new thought poster instance
func NewOriginalThoughtPoster(thoughtGen thoughts.OriginalThoughtGenerator, twitterClient *twitter.TwitterClient) *OriginalThoughtPoster {
	return &OriginalThoughtPoster{
		thoughtGen: thoughtGen,
		twitter:    twitterClient,
	}
}

// PostOriginalThought generates a thought and posts it to Twitter
func (p *OriginalThoughtPoster) PostOriginalThought(ctx context.Context, config OriginalThoughtConfig) (*twitter.Tweet, error) {
	// Generate the thought using base personality traits
	thought, err := p.thoughtGen.GenerateOriginalThought(ctx, thoughts.OriginalThoughtConfig{
		Topic:       config.Topic,
		MaxLength:   MaxTweetLength,
		Temperature: config.Temperature,
		Personality: traits.BasePromptSections,
	})
	if err != nil {
		return nil, fmt.Errorf("error generating thought: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"thought": thought,
		"topic":   config.Topic,
		"length":  len(thought),
	}).Debug("generated thought for tweet")

	// Post the thought to Twitter
	tweet, err := p.twitter.PostTweet(ctx, thought, &twitter.TweetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error posting tweet: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"tweet_id": tweet.ID,
		"text":     tweet.Text,
	}).Info("successfully posted thought to Twitter")

	return tweet, nil
}

// ThoughtOptions configures the original thought posting action
type ThoughtOptions struct {
	Interval    time.Duration
	Topic       string  // Default topic to post about
	Temperature float64 // Controls randomness of thought generation
}

type OriginalThoughtAction struct {
	poster   *OriginalThoughtPoster
	options  ThoughtOptions
	stopChan chan struct{}
	logger   *logrus.Logger
}

func NewOriginalThoughtAction(
	thoughtGen thoughts.OriginalThoughtGenerator,
	twitterClient *twitter.TwitterClient,
	logger *logrus.Logger,
	options ThoughtOptions,
) *OriginalThoughtAction {
	return &OriginalThoughtAction{
		poster:   NewOriginalThoughtPoster(thoughtGen, twitterClient),
		options:  options,
		stopChan: make(chan struct{}),
		logger:   logger,
	}
}

func (a *OriginalThoughtAction) Name() string {
	return "original_thought_poster"
}

func (a *OriginalThoughtAction) Execute(ctx context.Context) error {
	ticker := time.NewTicker(a.options.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.stopChan:
			return nil
		case <-ticker.C:
			_, err := a.poster.PostOriginalThought(ctx, OriginalThoughtConfig{
				Topic:       a.options.Topic,
				Temperature: a.options.Temperature,
			})
			if err != nil {
				a.logger.WithError(err).Error("Failed to post original thought")
			}
		}
	}
}

func (a *OriginalThoughtAction) Stop() {
	close(a.stopChan)
}
