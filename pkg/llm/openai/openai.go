package openai

import (
	"context"
	"fmt"

	"github.com/brendanplayford/agent-go/pkg/llm"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Client struct {
	logger *logrus.Logger
	llm    llms.LLM
	config *Config
}

func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	llm, err := openai.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenAI: %w", err)
	}

	return &Client{
		logger: config.Logger,
		llm:    llm,
		config: config,
	}, nil
}

func (c *Client) Generate(ctx context.Context, prompt string, opts ...llm.Option) (string, error) {
	options := &llm.Options{
		Temperature: 0.7,
		MaxTokens:   1000,
		Model:       "gpt-4",
	}

	for _, opt := range opts {
		opt(options)
	}

	c.logger.WithFields(logrus.Fields{
		"temperature": options.Temperature,
		"maxTokens":   options.MaxTokens,
		"model":       options.Model,
	}).Debug("Generating completion")

	completion, err := c.llm.Call(ctx, prompt,
		llms.WithTemperature(options.Temperature),
		llms.WithMaxTokens(options.MaxTokens),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	return completion, nil
}
