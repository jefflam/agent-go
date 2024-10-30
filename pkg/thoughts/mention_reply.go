package thoughts

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	langchainprompts "github.com/tmc/langchaingo/prompts"
)

// MentionReplyConfig holds configuration for reply generation
type MentionReplyConfig struct {
	OriginalTweetText string
	ConversationID    string
	MaxLength         int
	Temperature       float64
	PromptSections    map[string]string
}

// MentionReplyGenerator defines the interface for generating replies
type MentionReplyGenerator interface {
	GenerateReply(ctx context.Context, config MentionReplyConfig) (string, error)
}

// DefaultMentionReplyGenerator implements the MentionReplyGenerator interface
type DefaultMentionReplyGenerator struct {
	llm llms.Model
}

// NewMentionReplyGenerator creates a new reply generator instance
func NewMentionReplyGenerator(llm llms.Model) MentionReplyGenerator {
	return &DefaultMentionReplyGenerator{
		llm: llm,
	}
}

// GenerateReply creates a reply based on the original tweet and personality
func (g *DefaultMentionReplyGenerator) GenerateReply(ctx context.Context, config MentionReplyConfig) (string, error) {
	replyPrompt := langchainprompts.NewPromptTemplate(
		`You are responding to a tweet. Here is your personality:

{{.personality}}

Original Tweet: {{.originalTweet}}
Conversation ID: {{.conversationID}}

Your reply (must be under {{.maxLength}} characters):`,
		[]string{"personality", "originalTweet", "conversationID", "maxLength"},
	)

	// Format all personality sections into a single string
	var personalityText strings.Builder
	for section, content := range config.PromptSections {
		personalityText.WriteString(fmt.Sprintf("\n%s:\n%s\n", section, content))
	}

	formattedPrompt, err := replyPrompt.Format(map[string]any{
		"personality":    personalityText.String(),
		"originalTweet":  config.OriginalTweetText,
		"conversationID": config.ConversationID,
		"maxLength":      config.MaxLength,
	})
	if err != nil {
		return "", fmt.Errorf("error formatting reply prompt: %w", err)
	}

	reply, err := g.llm.Call(ctx, formattedPrompt)
	if err != nil {
		return "", fmt.Errorf("error generating reply: %w", err)
	}

	return reply, nil
}
