package agent

import (
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// Agent represents a Twitter AI agent that monitors mentions
type Agent struct {
	client   *twitter.TwitterClient
	llm      llms.Model
	logger   *logrus.Logger
	interval time.Duration
	ticker   *time.Ticker
}

// Config holds the configuration for the Agent
type Config struct {
	LLM           llms.Model
	Logger        *logrus.Logger
	TwitterClient *twitter.TwitterClient
	Interval      time.Duration
}
