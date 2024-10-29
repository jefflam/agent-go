package agentconfig

import (
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
	mentionsAction := actions.NewMentionsHandler(
		config.TwitterClient,
		config.LLM,
		config.Logger,
		actions.MentionsOptions{
			Interval:   30 * time.Second,
			MaxResults: 100,
		},
	)

	thoughtAction := actions.NewOriginalThoughtAction(
		thoughts.NewOriginalThoughtGenerator(config.LLM),
		config.TwitterClient,
		30*time.Second,
		config.Logger,
	)

	return []actions.Action{
		mentionsAction,
		thoughtAction,
	}, nil
}
