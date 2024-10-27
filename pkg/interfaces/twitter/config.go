package twitter

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type TwitterConfig struct {
	// API Authentication
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
	BearerToken       string // Optional for read-only operations

	// API Endpoints
	BaseURL          string
	TweetEndpoint    string
	UserEndpoint     string
	StreamEndpoint   string
	SearchEndpoint   string
	TimelineEndpoint string

	// Rate Limiting
	RateLimit     int
	RateWindow    int
	RetryAttempts int

	// API Fields Configuration (based on Twitter v2 data dictionary)
	DefaultFields   []string
	MetricFields    []string
	ExpansionFields []string

	// General Config
	Logger *logrus.Logger
}

func NewTwitterConfig() (*TwitterConfig, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Load rate limiting from env or use defaults
	rateLimit, _ := strconv.Atoi(getEnvOrDefault("TWITTER_RATE_LIMIT", "180"))
	rateWindow, _ := strconv.Atoi(getEnvOrDefault("TWITTER_RATE_WINDOW", "15"))
	retryAttempts, _ := strconv.Atoi(getEnvOrDefault("TWITTER_RETRY_ATTEMPTS", "3"))

	config := &TwitterConfig{
		// API Authentication
		ConsumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
		AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
		BearerToken:       os.Getenv("TWITTER_BEARER_TOKEN"),

		// API Endpoints
		BaseURL:          getEnvOrDefault("TWITTER_API_BASE_URL", "https://api.twitter.com/2"),
		TweetEndpoint:    "/tweets",
		UserEndpoint:     "/users",
		StreamEndpoint:   "/tweets/search/stream",
		SearchEndpoint:   "/tweets/search/recent",
		TimelineEndpoint: "/users/:id/tweets",

		// Rate Limiting
		RateLimit:     rateLimit,
		RateWindow:    rateWindow,
		RetryAttempts: retryAttempts,

		// Default API Fields (based on Twitter v2 data dictionary)
		DefaultFields: []string{"id", "text", "created_at"},
		MetricFields: []string{
			"impression_count",
			"like_count",
			"reply_count",
			"retweet_count",
			"quote_count",
		},
		ExpansionFields: []string{
			"author_id",
			"referenced_tweets.id",
			"in_reply_to_user_id",
			"attachments.media_keys",
			"attachments.poll_ids",
			"geo.place_id",
			"entities.mentions.username",
		},

		Logger: logrus.New(),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *TwitterConfig) Validate() error {
	// Validate logger
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	// For write operations (tweets), validate OAuth 1.0a credentials
	if c.ConsumerKey == "" || c.ConsumerSecret == "" ||
		c.AccessToken == "" || c.AccessTokenSecret == "" {
		c.Logger.Warn("OAuth 1.0a credentials not provided - write operations will not be available")

		// If no OAuth credentials, require bearer token for read-only operations
		if c.BearerToken == "" {
			return fmt.Errorf("either OAuth 1.0a credentials or Bearer token must be provided")
		}
	}

	// Validate rate limiting
	if c.RateLimit < 1 {
		return fmt.Errorf("rate limit must be positive")
	}
	if c.RateWindow < 1 {
		return fmt.Errorf("rate window must be positive")
	}
	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}

	// Set default endpoints if not provided
	if c.BaseURL == "" {
		c.BaseURL = "https://api.twitter.com/2"
	}
	if c.TweetEndpoint == "" {
		c.TweetEndpoint = "/tweets"
	}

	return nil
}

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEndpoint returns the full URL for a given endpoint
func (c *TwitterConfig) GetEndpoint(endpoint string) string {
	return c.BaseURL + endpoint
}

// GetTweetFields returns the default tweet fields plus any additional fields
func (c *TwitterConfig) GetTweetFields(additionalFields ...string) []string {
	fields := append([]string{}, c.DefaultFields...)
	return append(fields, additionalFields...)
}

// GetExpansions returns the configured expansion fields
func (c *TwitterConfig) GetExpansions() []string {
	return c.ExpansionFields
}

// HasWriteAccess returns true if OAuth 1.0a credentials are configured
func (c *TwitterConfig) HasWriteAccess() bool {
	return c.ConsumerKey != "" && c.ConsumerSecret != "" &&
		c.AccessToken != "" && c.AccessTokenSecret != ""
}

// HasReadAccess returns true if either OAuth 1.0a or Bearer token is configured
func (c *TwitterConfig) HasReadAccess() bool {
	return c.HasWriteAccess() || c.BearerToken != ""
}
