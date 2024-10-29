package agentconfig

import (
	"fmt"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/actions"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/lisanmuaddib/agent-go/pkg/thoughts"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

type ActionConfig struct {
	TwitterClient *twitter.TwitterClient
	LLM           llms.Model
	Logger        *logrus.Logger
}

// ConfigureActions sets up all agent actions
func ConfigureActions(config ActionConfig) ([]actions.Action, error) {
	mentionsHandler, err := actions.NewMentionsHandler(
		config.TwitterClient,
		config.LLM,
		config.Logger,
		actions.MentionsOptions{
			Interval:   30 * time.Second,
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
			Interval: 30 * time.Minute,
		},
	)

	return []actions.Action{
		mentionsHandler,
		thoughtAction,
	}, nil
}
