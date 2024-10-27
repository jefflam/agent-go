package llm

import (
	"context"
)

// LLM defines the interface for language model interactions
type LLM interface {
	Generate(ctx context.Context, prompt string, opts ...Option) (string, error)
}

// Option defines functional options for LLM configuration
type Option func(*Options)

// Options holds configuration for LLM calls
type Options struct {
	Temperature float64
	MaxTokens   int
	Model       string
}

// WithTemperature sets the temperature for generation
func WithTemperature(temp float64) Option {
	return func(o *Options) {
		o.Temperature = temp
	}
}

// WithMaxTokens sets the maximum tokens for generation
func WithMaxTokens(tokens int) Option {
	return func(o *Options) {
		o.MaxTokens = tokens
	}
}

// WithModel sets the model to use
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}
