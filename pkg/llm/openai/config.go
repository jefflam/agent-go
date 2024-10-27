package openai

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type Config struct {
	APIKey string
	Logger *logrus.Logger
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	return nil
}
