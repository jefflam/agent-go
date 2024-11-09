package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/interfaces/twitter"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Category        TweetCategory    `json:"category" gorm:"column:category"`
	ProcessedAt     time.Time        `json:"processed_at" gorm:"column:processed_at"`
	LastUpdated     time.Time        `json:"last_updated" gorm:"column:last_updated"`
	ProcessCount    int              `json:"process_count" gorm:"column:process_count"`
	ConversationRef *ConversationRef `json:"conversation_ref" gorm:"column:conversation_ref;type:jsonb"`
	AuthorName      string           `json:"author_name" gorm:"column:author_name"`
	AuthorUsername  string           `json:"author_username" gorm:"column:author_username"`
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
	mu     sync.RWMutex
	logger *logrus.Logger
	db     *gorm.DB
}

func NewTweetStore(logger *logrus.Logger, db *gorm.DB) (*TweetStore, error) {
	return &TweetStore{
		logger: logger,
		db:     db,
	}, nil
}

func (s *TweetStore) SaveTweet(tweet twitter.Tweet, category TweetCategory, authorName, authorUsername string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"tweet_id":        tweet.ID,
		"conversation_id": tweet.ConversationID,
		"text":            tweet.Text,
		"category":        category,
		"author_id":       tweet.AuthorID,
		"author_name":     authorName,
		"author_username": authorUsername,
	}).Debug("Attempting to save tweet")

	now := time.Now()

	// Prepare conversation ref if exists
	var conversationRef *ConversationRef
	if tweet.ConversationID != "" {
		s.logger.WithFields(logrus.Fields{
			"tweet_id":        tweet.ID,
			"conversation_id": tweet.ConversationID,
		}).Debug("Processing tweet with conversation ID")

		conversationRef = &ConversationRef{
			ConversationID: tweet.ConversationID,
			LastReplyAt:    now,
		}

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
			conversationRef.IsRoot = true
			conversationRef.RootID = tweet.ID
			s.logger.Debug("Tweet marked as conversation root")
		}
	}

	// Convert to database model
	storedTweet := StoredTweet{
		Tweet:           tweet,
		Category:        category,
		ProcessedAt:     now,
		LastUpdated:     now,
		AuthorName:      authorName,
		AuthorUsername:  authorUsername,
		ConversationRef: conversationRef,
	}

	// Upsert the tweet
	result := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"text",
			"conversation_id",
			"author_name",
			"author_username",
			"category",
			"last_updated",
			"conversation_ref",
			"process_count",
			"attachments",
			"context_annotations",
			"edit_controls",
			"edit_history_tweet_ids",
			"entities",
			"geo",
			"lang",
			"possibly_sensitive",
			"public_metrics",
			"referenced_tweets",
			"reply_settings",
			"source",
			"withheld",
		}),
	}).Create(&storedTweet)

	if result.Error != nil {
		return fmt.Errorf("failed to save tweet: %w", result.Error)
	}

	s.logger.WithFields(logrus.Fields{
		"tweet_id":          tweet.ID,
		"conversation_id":   tweet.ConversationID,
		"category":          category,
		"public_metrics":    tweet.PublicMetrics,
		"referenced_tweets": tweet.ReferencedTweets,
		"author_name":       storedTweet.AuthorName,
		"author_username":   storedTweet.AuthorUsername,
	}).Info("Successfully saved tweet to database")

	return nil
}

func (s *TweetStore) GetTweet(id string) (*StoredTweet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tweet StoredTweet
	result := s.db.Where("id = ?", id).First(&tweet)
	if result.Error != nil {
		return nil, fmt.Errorf("tweet not found: %s", id)
	}

	return &tweet, nil
}

func (s *TweetStore) GetTweetsByCategory(category TweetCategory) []StoredTweet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tweets []StoredTweet
	s.db.Where("category = ?", category).Find(&tweets)
	return tweets
}

func (s *TweetStore) GetConversation(conversationID string) []StoredTweet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tweets []StoredTweet
	s.db.Where("conversation_id = ?", conversationID).Find(&tweets)
	return tweets
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
