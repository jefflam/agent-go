package agentconfig

import (
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/actions"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/memory"
	"github.com/lisanmuaddib/agent-go/pkg/thoughts"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// Action timing constants define intervals for various agent activities
const (
	// MentionsCheckInterval is how often the agent checks for new mentions
	// Example: MentionsCheckInterval = 2 * time.Minute
	MentionsCheckInterval = 120 * time.Second

	// OriginalThoughtInterval is how often the agent generates and posts original thoughts
	// Example: OriginalThoughtInterval = 30 * time.Minute
	OriginalThoughtInterval = 30 * time.Minute

	// TweetResponseInterval is how often the agent processes and responds to pending tweets
	// Example: TweetResponseInterval = 5 * time.Minute
	TweetResponseInterval = 15 * time.Second
)

type ActionConfig struct {
	TwitterClient *twitter.TwitterClient
	LLM           llms.Model
	Logger        *logrus.Logger
	TweetStore    *memory.TweetStore
}

// ConfigureActions sets up all agent actions
func ConfigureActions(config ActionConfig) ([]actions.Action, error) {
	mentionsHandler, err := actions.NewMentionsHandler(
		config.TwitterClient,
		config.LLM,
		config.Logger,
		config.TweetStore,
		actions.MentionsOptions{
			Interval:   MentionsCheckInterval,
			MaxResults: 100,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mentions handler: %w", err)
	}

	thoughtAction := actions.NewOriginalThoughtAction(
		thoughts.NewOriginalThoughtGenerator(config.LLM),
		config.TwitterClient,
		config.Logger,
		actions.ThoughtOptions{
			Interval: OriginalThoughtInterval,
		},
	)

	replyGenerator := thoughts.NewMentionReplyGenerator(config.LLM)

	tweetResponder := actions.NewTweetResponder(
		config.TweetStore,
		config.TwitterClient,
		config.Logger,
		replyGenerator,
	)

	tweetResponseAction := actions.NewTweetResponseAction(
		tweetResponder,
		config.Logger,
		actions.TweetResponseOptions{
			Interval:    TweetResponseInterval,
			BatchConfig: actions.DefaultBatchConfig(),
		},
	)

	return []actions.Action{
		mentionsHandler,
		thoughtAction,
		tweetResponseAction,
	}, nil
}
