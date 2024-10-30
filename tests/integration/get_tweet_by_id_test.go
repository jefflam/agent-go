package integration

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

const (
	testTweetID = "1851403414191689969"
)

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

var _ = Describe("GetTweetByID", func() {
	var (
		client *twitter.TwitterClient
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
			DefaultFields: []string{
				"id",
				"text",
				"created_at",
				"conversation_id",
				"in_reply_to_user_id",
				"referenced_tweets",
				"author_id",
			},
			MetricFields: []string{
				"like_count",
				"reply_count",
				"retweet_count",
			},
			ExpansionFields: []string{
				"author_id",
				"referenced_tweets.id",
				"in_reply_to_user_id",
			},
		}

		var err error
		client, err = twitter.NewTwitterClient(config)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	})

	AfterEach(func() {
		cancel()
	})

	Context("when fetching a specific tweet", func() {
		It("should successfully get tweet with ID "+testTweetID, func() {
			params := twitter.GetTweetByIDParams{
				TweetID: testTweetID,
			}

			dataChan, errChan := client.GetTweetByID(ctx, params)
			var receivedData bool

			for {
				select {
				case resp, ok := <-dataChan:
					if !ok {
						dataChan = nil
						continue
					}
					receivedData = true
					Expect(resp).NotTo(BeNil())

					tweet, err := resp.UnmarshalTweet()
					Expect(err).NotTo(HaveOccurred())

					// Verify tweet details
					Expect(tweet.ID).To(Equal(testTweetID))

					// Log tweet details for debugging before assertion
					logger.WithFields(logrus.Fields{
						"tweet_id":          tweet.ID,
						"conversation_id":   tweet.ConversationID,
						"in_reply_to_user":  tweet.InReplyToUserID,
						"referenced_tweets": tweet.ReferencedTweets,
						"text":              tweet.Text,
					}).Info("Tweet details")

					// This tweet should have a conversation_id since it's a reply
					Expect(tweet.ConversationID).NotTo(BeEmpty(),
						"Tweet %s is a reply (has in_reply_to_user_id or referenced_tweets) but missing conversation_id",
						tweet.ID)

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
