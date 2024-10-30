package thoughts

import (
	"context"
	"fmt"
	"strings"

	"github.com/lisanmuaddib/agent-go/internal/personality/traits"
	"github.com/tmc/langchaingo/llms"
	langchainprompts "github.com/tmc/langchaingo/prompts"
)

// DefaultReplyPersonality provides the base personality traits for replies
var DefaultReplyPersonality = traits.BasePromptSections

// MentionReplyConfig holds configuration for reply generation
type MentionReplyConfig struct {
	TweetText   string
	MaxLength   int
	Temperature float64
	Personality map[string]string // Optional: will use DefaultReplyPersonality if nil
}

type MentionReplyGenerator interface {
	GenerateReply(ctx context.Context, config MentionReplyConfig) (string, error)
}

type DefaultMentionReplyGenerator struct {
	llm llms.Model
}

func NewMentionReplyGenerator(llm llms.Model) MentionReplyGenerator {
	return &DefaultMentionReplyGenerator{
		llm: llm,
	}
}

// GenerateReply creates a reply based on the tweet and personality
func (g *DefaultMentionReplyGenerator) GenerateReply(ctx context.Context, config MentionReplyConfig) (string, error) {
	personality := config.Personality
	if personality == nil {
		personality = DefaultReplyPersonality
	}

	replyPrompt := langchainprompts.NewPromptTemplate(
		`You are responding to a tweet. Here is your personality:

{{.personality}}

Tweet to respond to: {{.tweet}}

Requirements:
1. Your reply MUST be under {{.maxLength}} characters
2. Stay in character
3. Be engaging and memorable
4. Respond directly to the tweet's content

Your reply:`,
		[]string{"personality", "tweet", "maxLength"},
	)

	// Format personality traits into a string
	var personalityText strings.Builder
	for section, content := range personality {
		personalityText.WriteString(fmt.Sprintf("\n%s:\n%s\n", section, content))
	}

	formattedPrompt, err := replyPrompt.Format(map[string]any{
		"personality": personalityText.String(),
		"tweet":       config.TweetText,
		"maxLength":   config.MaxLength,
	})
	if err != nil {
		return "", fmt.Errorf("error formatting reply prompt: %w", err)
	}

	reply, err := g.llm.Call(ctx, formattedPrompt,
		llms.WithTemperature(config.Temperature),
		llms.WithMaxTokens(config.MaxLength),
	)
	if err != nil {
		return "", fmt.Errorf("error generating reply: %w", err)
	}

	return reply, nil
}
