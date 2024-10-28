package thoughts

import (
	"context"
	"fmt"

	"github.com/lisanmuaddib/agent-go/pkg/llm"
	langchainprompts "github.com/tmc/langchaingo/prompts"
)

// OriginalThoughtConfig holds configuration for thought generation
type OriginalThoughtConfig struct {
	Topic       string
	MaxLength   int
	Temperature float64
	Personality map[string]string
}

// OriginalThoughtGenerator defines the interface for generating thoughts
type OriginalThoughtGenerator interface {
	GenerateOriginalThought(ctx context.Context, config OriginalThoughtConfig) (string, error)
}

// DefaultOriginalThoughtGenerator implements the OriginalThoughtGenerator interface
type DefaultOriginalThoughtGenerator struct {
	llm llm.LLM
}

// NewOriginalThoughtGenerator creates a new thought generator instance
func NewOriginalThoughtGenerator(llm llm.LLM) OriginalThoughtGenerator {
	return &DefaultOriginalThoughtGenerator{
		llm: llm,
	}
}

// GenerateOriginalThought creates a new thought based on the provided configuration
func (g *DefaultOriginalThoughtGenerator) GenerateOriginalThought(ctx context.Context, config OriginalThoughtConfig) (string, error) {
	// Create a base prompt template for thought generation
	thoughtPrompt := langchainprompts.NewPromptTemplate(
		`Generate a single thought (maximum {{.maxLength}} characters) that reflects the following personality traits:

{{.personality}}

The thought should be about: {{.topic}}

Requirements:
1. Stay within character
2. Be concise and impactful
3. Maintain consistent tone
4. Be engaging and memorable

Generated thought:`,
		[]string{"personality", "topic", "maxLength"},
	)

	// Format personality traits into a string
	personalityStr := formatPersonalityTraits(config.Personality)

	// Format the prompt with configuration
	formattedPrompt, err := thoughtPrompt.Format(map[string]any{
		"personality": personalityStr,
		"topic":       config.Topic,
		"maxLength":   config.MaxLength,
	})
	if err != nil {
		return "", fmt.Errorf("error formatting thought prompt: %w", err)
	}

	// Generate the thought using the LLM
	thought, err := g.llm.Generate(ctx, formattedPrompt,
		llm.WithTemperature(config.Temperature),
		llm.WithMaxTokens(config.MaxLength),
	)
	if err != nil {
		return "", fmt.Errorf("error generating thought: %w", err)
	}

	return thought, nil
}

// formatPersonalityTraits converts personality map to formatted string
func formatPersonalityTraits(traits map[string]string) string {
	var result string
	for category, trait := range traits {
		result += fmt.Sprintf("%s:\n%s\n\n", category, trait)
	}
	return result
}
