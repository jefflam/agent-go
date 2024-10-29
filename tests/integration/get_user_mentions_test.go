package integration

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	. "github.com/onsi/ginkgo/v2" // Only import v2
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

var _ = Describe("GetUserMentions", func() {
	var (
		client *twitter.TwitterClient
		userID string
		logger *logrus.Logger
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		// Skip if not running integration tests
		if os.Getenv("INTEGRATION_TESTS") != "true" {
			Skip("Skipping integration test")
		}

		// Setup logger
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Get required environment variables
		bearerToken := os.Getenv("TWITTER_BEARER_TOKEN")
		Expect(bearerToken).NotTo(BeEmpty(), "TWITTER_BEARER_TOKEN environment variable is required")

		// Initialize config
		config := &twitter.TwitterConfig{
			BearerToken:       bearerToken,
			ConsumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
			ConsumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
			AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
			AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
			BaseURL:           "https://api.twitter.com/2",
			RateLimit:         180,
			RateWindow:        int(15 * time.Minute / time.Second),
			Logger:            logger,
			DefaultFields:     []string{"id", "text", "created_at"},
			MetricFields:      []string{"like_count", "reply_count", "retweet_count"},
			ExpansionFields: []string{
				"author_id",
				"referenced_tweets.id",
				"in_reply_to_user_id",
			},
		}

		var err error
		client, err = twitter.NewTwitterClient(config)
		Expect(err).NotTo(HaveOccurred())

		// Get authenticated user ID
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		userID, err = client.GetAuthenticatedUserID(ctx)
		Expect(err).NotTo(HaveOccurred(), "Failed to get authenticated user ID")
	})

	AfterEach(func() {
		cancel()
	})

	Context("when fetching user mentions", func() {
		It("should successfully get authenticated user mentions", func() {
			params := twitter.GetUserMentionsParams{
				MaxResults: 10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dataChan, errChan := client.GetUserMentions(ctx, params)
			var receivedData bool

			for {
				select {
				case resp, ok := <-dataChan:
					if !ok {
						dataChan = nil
						continue
					}
					receivedData = true
					if resp != nil && len(resp.Data) > 0 {
						Expect(resp.Meta).NotTo(BeNil())
						tweets, err := resp.UnmarshalTweets()
						Expect(err).NotTo(HaveOccurred())
						if len(tweets) > 0 {
							Expect(tweets[0].ID).NotTo(BeEmpty())
							Expect(tweets[0].Text).NotTo(BeEmpty())
							GinkgoWriter.Printf("Received mention: %s\n", tweets[0].Text)
							GinkgoWriter.Printf("Conversation ID: %s\n", tweets[0].ConversationID)
						}
					}
				case err, ok := <-errChan:
					if !ok {
						errChan = nil
						continue
					}
					Expect(err).NotTo(HaveOccurred())
				case <-ctx.Done():
					return
				}

				if dataChan == nil && errChan == nil {
					break
				}
			}

			Expect(receivedData).To(BeTrue(), "Should have received data")
		})

		It("should return error for invalid max results", func() {
			params := twitter.GetUserMentionsParams{
				MaxResults: 101,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dataChan, errChan := client.GetUserMentions(ctx, params)
			var lastError error

			for {
				select {
				case _, ok := <-dataChan:
					if !ok {
						dataChan = nil
					}
				case err, ok := <-errChan:
					if !ok {
						errChan = nil
						continue
					}
					lastError = err
				case <-ctx.Done():
					return
				}

				if dataChan == nil && errChan == nil {
					break
				}
			}

			Expect(lastError).To(HaveOccurred())
			Expect(lastError.Error()).To(ContainSubstring("invalid max_results"))
		})

		It("should successfully get mentions with specific user ID", func() {
			params := twitter.GetUserMentionsParams{
				UserID:     userID,
				MaxResults: 10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dataChan, errChan := client.GetUserMentions(ctx, params)
			var receivedData bool

			for {
				select {
				case resp, ok := <-dataChan:
					if !ok {
						dataChan = nil
						continue
					}
					receivedData = true
					if resp != nil && len(resp.Data) > 0 {
						Expect(resp.Meta).NotTo(BeNil())
						tweets, err := resp.UnmarshalTweets()
						Expect(err).NotTo(HaveOccurred())
						if len(tweets) > 0 {
							Expect(tweets[0].ID).NotTo(BeEmpty())
							Expect(tweets[0].Text).NotTo(BeEmpty())
						}
					}
				case err, ok := <-errChan:
					if !ok {
						errChan = nil
						continue
					}
					Expect(err).NotTo(HaveOccurred())
				case <-ctx.Done():
					return
				}

				if dataChan == nil && errChan == nil {
					break
				}
			}

			Expect(receivedData).To(BeTrue(), "Should have received data")
		})
	})
})
