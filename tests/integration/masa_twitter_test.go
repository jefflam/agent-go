package integration

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/pkg/masa/masatwitter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

const (
	// SearchOptions defines default search parameters
	DefaultTweetCount     = 10
	DefaultRequestCount   = 1
	DefaultRequestDelay   = 2 * time.Second
	DefaultSearchQuery    = "#bitcoin"
	DefaultAPIEndpoint    = "http://localhost:8080/api/v1/data/twitter/tweets/recent"
	DefaultRequestTimeout = 120 * time.Second
)

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

var _ = Describe("TwitterScraper", func() {
	var (
		twitterClient *masatwitter.Client
		logger        *logrus.Logger
	)

	BeforeEach(func() {
		// Skip if not running integration tests
		if os.Getenv("INTEGRATION_TESTS") != "true" {
			Skip("Skipping integration test")
		}

		// Setup logger
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Initialize Twitter client config
		clientConfig := &masatwitter.Config{
			APIEndpoint:      DefaultAPIEndpoint,
			RequestTimeout:   DefaultRequestTimeout,
			TweetsPerRequest: DefaultTweetCount,
			Logger:           logger,
		}

		// Override with environment variable if set
		if envEndpoint := os.Getenv("MASA_TWITTER_API_ENDPOINT"); envEndpoint != "" {
			clientConfig.APIEndpoint = envEndpoint
		}

		twitterClient = masatwitter.NewClient(clientConfig)
		Expect(twitterClient).NotTo(BeNil())
	})

	It("should fetch golang tweets", func() {
		tweets, err := twitterClient.SearchWithOptions(DefaultSearchQuery, masatwitter.SearchOptions{
			TweetCount: DefaultTweetCount,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(tweets).NotTo(BeNil())
		Expect(tweets).To(HaveLen(DefaultTweetCount))

		// Convert response to pretty JSON for logging
		jsonData, err := json.MarshalIndent(tweets, "", "    ")
		Expect(err).NotTo(HaveOccurred())

		// Log the raw JSON response
		logger.WithFields(logrus.Fields{
			"query":        DefaultSearchQuery,
			"tweet_count":  DefaultTweetCount,
			"raw_response": string(jsonData),
		}).Info("Received tweet data")
	})
})
