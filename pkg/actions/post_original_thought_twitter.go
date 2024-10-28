package actions

import (
	"context"
	"fmt"

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
