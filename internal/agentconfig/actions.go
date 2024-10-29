package agentconfig

import (
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/actions"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
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

	return []actions.Action{mentionsAction}, nil
}
