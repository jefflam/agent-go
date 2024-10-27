package thoughts

import (
	"context"
	"fmt"

	"github.com/brendanplayford/agent-go/internal/personality/traits"
	"github.com/brendanplayford/agent-go/pkg/llm"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Processor struct {
	logger *logrus.Logger
	llm    llm.LLM
}

func NewProcessor(config Config) (*Processor, error) {
	llmClient, err := openai.NewClient(&openai.Config{
		APIKey: config.OpenAIKey,
		Logger: config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenAI client: %w", err)
	}

	return &Processor{
		logger: config.Logger,
		llm:    llmClient,
	}, nil
}

func (p *Processor) Process(ctx context.Context, input string) (*Thought, error) {
	thought := &Thought{
		ID:       NewThoughtID(),
		Content:  input,
		Status:   ThoughtStatusPending,
		Metadata: make(map[string]interface{}),
	}

	// Create prompt template using base personality
	prompt := traits.NewAgentPrompt(
		[]string{"tweet_composer"},
		"tweet_composer: Use this tool to compose tweets in your unique voice",
	)

	// Format the prompt with input
	formattedPrompt, err := prompt.Format(map[string]any{
		"input": fmt.Sprintf("Compose a tweet about: %s", input),
	})
	if err != nil {
		thought.Status = ThoughtStatusFailed
		return thought, fmt.Errorf("failed to format prompt: %w", err)
	}

	// Call OpenAI
	completion, err := p.llm.Call(ctx, formattedPrompt,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(280), // Twitter's max length
	)
	if err != nil {
		thought.Status = ThoughtStatusFailed
		return thought, fmt.Errorf("failed to generate thought: %w", err)
	}

	thought.Content = completion
	thought.Status = ThoughtStatusProcessed
	thought.Metadata["prompt"] = formattedPrompt
	thought.Metadata["model"] = "gpt-4"

	p.logger.WithFields(logrus.Fields{
		"thoughtID": thought.ID,
		"status":    thought.Status,
		"content":   thought.Content,
	}).Info("Thought processed successfully")

	return thought, nil
}
