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
	TweetText           string            `json:"tweet_text"`
	ConversationContext string            `json:"conversation_context"` // Optional: for thread context
	MaxLength           int               `json:"max_length"`
	Temperature         float64           `json:"temperature"`
	AuthorUsername      string            `json:"author_username,omitempty"` // Optional: for better context
	AuthorName          string            `json:"author_name,omitempty"`     // Optional: for better context
	Category            string            `json:"category,omitempty"`        // Optional: type of interaction
	Language            string            `json:"language,omitempty"`        // Optional: for language support
	Personality         map[string]string // Optional: will use DefaultReplyPersonality if nil
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

	// Use enhanced prompt if conversation context is available
	var promptTemplate string
	if config.ConversationContext != "" {
		promptTemplate = conversationalReplyPrompt
	} else {
		promptTemplate = standardReplyPrompt
	}

	replyPrompt := langchainprompts.NewPromptTemplate(
		promptTemplate,
		[]string{"personality", "tweet", "maxLength", "context", "authorUsername", "authorName", "category"},
	)

	// Format personality traits into a string
	var personalityText strings.Builder
	for section, content := range personality {
		personalityText.WriteString(fmt.Sprintf("\n%s:\n%s\n", section, content))
	}

	// Prepare prompt data with optional fields
	promptData := map[string]any{
		"personality": personalityText.String(),
		"tweet":       config.TweetText,
		"maxLength":   config.MaxLength,
		"context":     config.ConversationContext,
	}

	// Add optional fields if present
	if config.AuthorUsername != "" {
		promptData["authorUsername"] = config.AuthorUsername
	}
	if config.AuthorName != "" {
		promptData["authorName"] = config.AuthorName
	}
	if config.Category != "" {
		promptData["category"] = config.Category
	}

	formattedPrompt, err := replyPrompt.Format(promptData)
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

// standardReplyPrompt is the original prompt template for backward compatibility
const standardReplyPrompt = `You are responding to a tweet. Here is your personality:

{{.personality}}

Tweet to respond to: {{.tweet}}

Requirements:
1. Your reply MUST be under {{.maxLength}} characters
2. Stay in character
3. Be engaging and memorable
4. Respond directly to the tweet's content

Your reply:`

// conversationalReplyPrompt is the enhanced prompt template for conversation context
const conversationalReplyPrompt = `You are responding to a tweet conversation. Here is your personality:

{{.personality}}

CONVERSATION CONTEXT:
{{.context}}

Tweet to respond to: {{.tweet}}
{{if .authorUsername}}From: @{{.authorUsername}}{{if .authorName}} ({{.authorName}}){{end}}{{end}}
{{if .category}}Interaction type: {{.category}}{{end}}

Requirements:
1. Your reply MUST be under {{.maxLength}} characters
2. Stay in character
3. Be engaging and memorable
4. Consider the full conversation context
5. Maintain conversation flow
6. Use appropriate emojis when relevant

Your reply:`
