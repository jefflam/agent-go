package integration

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

const (
	testUserID = "1848122847585017856" // Your actual user ID from tweets.json
)

// MockTwitterClient implements just the GetAuthenticatedUserID method
type MockTwitterClient struct{}

func (m *MockTwitterClient) GetAuthenticatedUserID(ctx context.Context) (string, error) {
	return testUserID, nil
}

var _ = Describe("RecallTweetsNeedingReply", func() {
	var (
		store  *memory.TweetStore
		logger *logrus.Logger
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		// Setup logger
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Change to project root directory for correct file access
		err := os.Chdir("../..")
		Expect(err).NotTo(HaveOccurred(), "Failed to change to project root directory")

		// Verify we can access the tweets file
		_, err = os.Stat("data/tweets/tweets.json")
		Expect(err).NotTo(HaveOccurred(), "tweets.json should exist")

		// Get and log absolute path for verification
		absPath, err := filepath.Abs("data/tweets/tweets.json")
		Expect(err).NotTo(HaveOccurred())
		logger.WithField("tweets_file", absPath).Info("Loading tweets from")

		// Initialize tweet store
		store, err = memory.NewTweetStore(logger)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	})

	AfterEach(func() {
		cancel()
	})

	Context("when checking stored tweets for needed replies", func() {
		It("should identify tweets needing replies", func() {
			client := &MockTwitterClient{}
			tweets, err := store.RecallTweetsNeedingReply(ctx, client)
			Expect(err).NotTo(HaveOccurred())

			// Log the results
			logger.WithFields(logrus.Fields{
				"tweets_found": len(tweets),
				"tweets":       tweets,
			}).Info("Found tweets needing reply")

			// Verify the results
			for _, tweet := range tweets {
				By("checking tweet properties")
				Expect(tweet.ConversationID).NotTo(BeEmpty())
				Expect(tweet.TweetID).NotTo(BeEmpty())

				By("verifying conversation participation")
				Expect(tweet.IsParticipating).To(BeTrue(),
					"Expected to be participating in conversation %s",
					tweet.ConversationID)

				By("checking unread replies")
				Expect(tweet.UnreadReplies).To(BeNumerically(">", 0),
					"Expected unread replies in conversation %s",
					tweet.ConversationID)

				// Get the actual tweet content
				storedTweet, err := store.GetTweet(tweet.TweetID)
				Expect(err).NotTo(HaveOccurred())

				logger.WithFields(logrus.Fields{
					"tweet_id":         tweet.TweetID,
					"conversation_id":  tweet.ConversationID,
					"unread_replies":   tweet.UnreadReplies,
					"last_reply_id":    tweet.LastReplyID,
					"last_reply_time":  tweet.LastReplyTime,
					"is_participating": tweet.IsParticipating,
					"text":             storedTweet.Text,
					"author_id":        storedTweet.AuthorID,
				}).Info("Tweet details")
			}
		})
	})
})
