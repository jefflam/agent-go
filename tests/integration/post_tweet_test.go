package twitter_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

var _ = Describe("PostTweet", func() {
	var (
		client *twitter.TwitterClient
		config *twitter.TwitterConfig
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetLevel(logrus.DebugLevel)

		config = &twitter.TwitterConfig{
			// API Authentication
			ConsumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
			ConsumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
			AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
			AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),

			// API Endpoints
			BaseURL:       "https://api.twitter.com/2",
			TweetEndpoint: "/tweets",

			// Rate Limiting - Add these required values
			RateLimit:     180, // Default Twitter API rate limit
			RateWindow:    15,  // 15-minute window
			RetryAttempts: 3,   // Number of retry attempts

			// Required logger
			Logger: logger,

			// Optional fields configuration
			DefaultFields: []string{"id", "text", "created_at"},
		}

		var err error
		client, err = twitter.NewTwitterClient(config)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("PostTweet", func() {
		It("should successfully post a tweet", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			tweetText := fmt.Sprintf("Test tweet %s", time.Now().Format(time.RFC3339))
			tweet, err := client.PostTweet(ctx, tweetText, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(tweet).NotTo(BeNil())
			Expect(tweet.Text).To(Equal(tweetText))
			Expect(tweet.ID).NotTo(BeEmpty())
		})

		It("should handle context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			tweet, err := client.PostTweet(ctx, "Test tweet", nil)
			Expect(err).To(Equal(context.Canceled))
			Expect(tweet).To(BeNil())
		})

		It("should handle context timeout", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()
			time.Sleep(time.Millisecond)
			tweet, err := client.PostTweet(ctx, "Test tweet", nil)
			Expect(err).To(Equal(context.DeadlineExceeded))
			Expect(tweet).To(BeNil())
		})
	})

	Context("PostReply", func() {
		var parentTweetID string

		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Add proper error handling and nil checks
			var err error
			var tweet *twitter.Tweet
			for attempts := 0; attempts < 3; attempts++ {
				tweet, err = client.PostTweet(ctx, "Parent tweet for testing replies", nil)
				if err == nil && tweet != nil {
					parentTweetID = tweet.ID
					break
				}
				time.Sleep(time.Second * 2)
			}
			// Change this to handle the error properly
			Expect(err).NotTo(HaveOccurred(), "Failed to create parent tweet after retries")
			Expect(tweet).NotTo(BeNil(), "Tweet response should not be nil")
			Expect(parentTweetID).NotTo(BeEmpty(), "Parent tweet ID should not be empty")
		})

		It("should successfully post a reply", func() {
			// Only run the test if we have a valid parent tweet
			if parentTweetID == "" {
				Skip("Parent tweet creation failed")
			}

			replyText := fmt.Sprintf("Test reply %s", time.Now().Format(time.RFC3339))
			replyOptions := &twitter.TweetOptions{
				ReplyTo: parentTweetID,
			}
			tweet, err := client.PostTweet(context.Background(), replyText, replyOptions)
			Expect(err).NotTo(HaveOccurred())
			Expect(tweet).NotTo(BeNil())
			Expect(tweet.Text).To(Equal(replyText))
		})
	})

	Context("PostQuote", func() {
		var tweetToQuoteID string

		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Add proper error handling and nil checks
			var err error
			var tweet *twitter.Tweet
			for attempts := 0; attempts < 3; attempts++ {
				tweet, err = client.PostTweet(ctx, "Tweet to be quoted", nil)
				if err == nil && tweet != nil {
					tweetToQuoteID = tweet.ID
					break
				}
				time.Sleep(time.Second * 2)
			}
			// Change this to handle the error properly
			Expect(err).NotTo(HaveOccurred(), "Failed to create tweet to quote after retries")
			Expect(tweet).NotTo(BeNil(), "Tweet response should not be nil")
			Expect(tweetToQuoteID).NotTo(BeEmpty(), "Tweet to quote ID should not be empty")
		})

		It("should successfully post a quote tweet", func() {
			// Only run the test if we have a valid tweet to quote
			if tweetToQuoteID == "" {
				Skip("Tweet to quote creation failed")
			}

			quoteText := fmt.Sprintf("Test quote %s", time.Now().Format(time.RFC3339))
			tweet, err := client.PostQuote(context.Background(), quoteText, tweetToQuoteID)
			Expect(err).NotTo(HaveOccurred())
			Expect(tweet).NotTo(BeNil())
			Expect(tweet.Text).To(HavePrefix(quoteText))
		})
	})
})
