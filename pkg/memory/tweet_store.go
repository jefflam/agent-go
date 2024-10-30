package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
)

// TweetCategory represents different types of tweets we store
type TweetCategory string

const (
	CategoryMention      TweetCategory = "mention"
	CategoryReply        TweetCategory = "reply"
	CategoryQuote        TweetCategory = "quote"
	CategoryRetweet      TweetCategory = "retweet"
	CategoryDM           TweetCategory = "dm"
	CategoryConversation TweetCategory = "conversation"
)

// StoredTweet extends the twitter.Tweet with processing metadata
type StoredTweet struct {
	twitter.Tweet                    // Embeds all standard Tweet fields (includes AuthorID, metrics, etc)
	Category        TweetCategory    `json:"category"`
	ProcessedAt     time.Time        `json:"processed_at"`
	LastUpdated     time.Time        `json:"last_updated"`
	ProcessCount    int              `json:"process_count"`
	ConversationRef *ConversationRef `json:"conversation_ref,omitempty"`
}

// ConversationRef holds metadata about the tweet's place in a conversation
type ConversationRef struct {
	IsRoot          bool      `json:"is_root"`           // Is this the first tweet in the conversation?
	ParentID        string    `json:"parent_id"`         // ID of the tweet this is replying to
	RootID          string    `json:"root_id"`           // ID of the first tweet in the conversation
	ConversationID  string    `json:"conversation_id"`   // Twitter's conversation ID
	ReplyDepth      int       `json:"reply_depth"`       // How deep in the reply chain
	LastReplyAt     time.Time `json:"last_reply_at"`     // When was the last reply in this conversation
	LastReplyID     string    `json:"last_reply_id"`     // ID of the last reply in the conversation
	LastReplyAuthor string    `json:"last_reply_author"` // AuthorID of the last reply
	ReplyCount      int       `json:"reply_count"`       // Number of replies in this conversation
}

type TweetStore struct {
	mu       sync.RWMutex
	filepath string
	logger   *logrus.Logger
	tweets   map[string]StoredTweet // Using tweet ID as key prevents duplicates
}

func NewTweetStore(logger *logrus.Logger) (*TweetStore, error) {
	filepath := "data/tweets/tweets.json"
	store := &TweetStore{
		filepath: filepath,
		logger:   logger,
		tweets:   make(map[string]StoredTweet),
	}
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data/tweets", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load existing tweets
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load tweet store: %w", err)
	}

	return store, nil
}

func (s *TweetStore) SaveTweet(tweet twitter.Tweet, category TweetCategory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"tweet_id":            tweet.ID,
		"raw_conversation_id": tweet.ConversationID,
		"text":                tweet.Text,
		"category":            category,
	}).Debug("Attempting to save tweet")

	now := time.Now()

	// Create stored tweet with all fields
	storedTweet := StoredTweet{
		Tweet:       tweet,
		Category:    category,
		ProcessedAt: now,
		LastUpdated: now,
	}

	// Add more detailed logging for conversation handling
	if tweet.ConversationID != "" {
		s.logger.WithFields(logrus.Fields{
			"tweet_id":        tweet.ID,
			"conversation_id": tweet.ConversationID,
		}).Debug("Processing tweet with conversation ID")

		conversationRef := &ConversationRef{
			ConversationID: tweet.ConversationID,
			LastReplyAt:    now,
		}

		// Check if this is a reply
		if tweet.ReferencedTweets != nil {
			s.logger.WithField("referenced_tweets", tweet.ReferencedTweets).Debug("Tweet has referenced tweets")
			for _, ref := range tweet.ReferencedTweets {
				if ref.Type == "replied_to" {
					conversationRef.ParentID = ref.ID
					conversationRef.IsRoot = false
					s.logger.WithFields(logrus.Fields{
						"parent_id": ref.ID,
						"type":      ref.Type,
					}).Debug("Found parent tweet reference")
				}
			}
		} else {
			// If no referenced tweets, this might be the root
			conversationRef.IsRoot = true
			conversationRef.RootID = tweet.ID
			s.logger.Debug("Tweet marked as conversation root")
		}

		storedTweet.ConversationRef = conversationRef
	} else {
		s.logger.WithField("tweet_id", tweet.ID).Debug("Tweet has no conversation ID")
	}

	// Verify the stored tweet before saving
	s.logger.WithFields(logrus.Fields{
		"tweet_id":         tweet.ID,
		"has_conversation": storedTweet.ConversationID != "",
		"has_conv_ref":     storedTweet.ConversationRef != nil,
		"category":         storedTweet.Category,
	}).Debug("About to store tweet")

	// Store the tweet
	s.tweets[tweet.ID] = storedTweet

	// Save to disk
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save tweet store: %w", err)
	}

	// Log successful save with full details
	s.logger.WithFields(logrus.Fields{
		"tweet_id":          tweet.ID,
		"conversation_id":   tweet.ConversationID,
		"category":          category,
		"public_metrics":    tweet.PublicMetrics,
		"referenced_tweets": tweet.ReferencedTweets,
		"stored_conv_id":    s.tweets[tweet.ID].ConversationID,
		"has_conv_ref":      s.tweets[tweet.ID].ConversationRef != nil,
	}).Info("Successfully saved tweet to store")

	return nil
}

func (s *TweetStore) GetTweet(id string) (*StoredTweet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tweet, ok := s.tweets[id]
	if !ok {
		return nil, fmt.Errorf("tweet not found: %s", id)
	}

	return &tweet, nil
}

func (s *TweetStore) GetTweetsByCategory(category TweetCategory) []StoredTweet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tweets []StoredTweet
	for _, tweet := range s.tweets {
		if tweet.Category == category {
			tweets = append(tweets, tweet)
		}
	}

	return tweets
}

func (s *TweetStore) GetConversation(conversationID string) []StoredTweet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var conversation []StoredTweet
	for _, tweet := range s.tweets {
		if tweet.ConversationID == conversationID {
			conversation = append(conversation, tweet)
		}
	}

	return conversation
}

func (s *TweetStore) load() error {
	s.logger.WithField("filepath", s.filepath).Debug("Loading tweets from file")

	data, err := os.ReadFile(s.filepath)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read tweets file")
		return err
	}

	s.logger.WithField("data_size", len(data)).Debug("Read tweets file")

	err = json.Unmarshal(data, &s.tweets)
	if err != nil {
		s.logger.WithError(err).Error("Failed to unmarshal tweets")
		return err
	}

	s.logger.WithField("tweets_loaded", len(s.tweets)).Info("Successfully loaded tweets from file")

	// Log a few tweet IDs for verification
	var tweetIDs []string
	for id := range s.tweets {
		tweetIDs = append(tweetIDs, id)
		if len(tweetIDs) >= 3 {
			break
		}
	}
	s.logger.WithField("sample_tweet_ids", tweetIDs).Debug("Sample of loaded tweets")

	return nil
}

func (s *TweetStore) save() error {
	data, err := json.MarshalIndent(s.tweets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tweets: %w", err)
	}

	tempFile := s.filepath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename for safer file writes
	if err := os.Rename(tempFile, s.filepath); err != nil {
		os.Remove(tempFile) // Clean up temp file if rename fails
		return fmt.Errorf("failed to save tweet store: %w", err)
	}

	return nil
}

// Helper method to determine tweet category based on tweet content
func DetermineTweetCategory(tweet twitter.Tweet) TweetCategory {
	if tweet.ConversationID != "" && tweet.ReferencedTweets != nil {
		for _, ref := range tweet.ReferencedTweets {
			if ref.Type == "replied_to" {
				return CategoryConversation
			}
		}
	}

	if tweet.ReferencedTweets != nil {
		for _, ref := range tweet.ReferencedTweets {
			switch ref.Type {
			case "quoted":
				return CategoryQuote
			case "retweeted":
				return CategoryRetweet
			}
		}
	}
	return CategoryMention
}
