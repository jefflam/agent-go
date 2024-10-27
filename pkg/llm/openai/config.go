package openai

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type OpenAIConfig struct {
	APIKey      string
	Logger      *logrus.Logger
	Temperature float64
	MaxTokens   int
	Model       string
}

// NewOpenAIConfig creates a new OpenAIConfig with OpenAI-specific values from environment variables
func NewOpenAIConfig() (*OpenAIConfig, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist in production
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	config := &OpenAIConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       os.Getenv("OPENAI_MODEL"),
		Temperature: 0.7,
		MaxTokens:   1000,
		Logger:      logrus.New(),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	// Set default values if not provided
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 1000
	}
	if c.Model == "" {
		c.Model = "gpt-4"
	}
	return nil
}
