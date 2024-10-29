package mentions

import (
	"context"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
)

// Processor handles the processing of Twitter mentions
type Processor struct {
	client *twitter.TwitterClient
	logger *logrus.Logger
}

// NewProcessor creates a new mention processor instance
func NewProcessor(client *twitter.TwitterClient, logger *logrus.Logger) *Processor {
	if logger == nil {
		logger = logrus.New()
	}

	return &Processor{
		client: client,
		logger: logger,
	}
}

// Process handles a single batch of mention processing
func (p *Processor) Process(ctx context.Context) error {
	// ... existing ProcessMentions code ...
}

func (p *Processor) processTweetResponse(ctx context.Context, resp *twitter.TweetResponse) {
	// ... existing processTweetResponse code ...
}
