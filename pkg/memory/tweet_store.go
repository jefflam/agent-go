package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"
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

// Add constants for bot details at package level
const (
	AgentUsername = "CatLordLaffy"
	AgentName     = "CatLordLaffy"
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

// TableName specifies the table name for GORM
func (StoredTweet) TableName() string {
	return "tweets"
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
	botID  string
	env    EnvConfig
}

func NewTweetStore(logger *logrus.Logger, db *gorm.DB, botID string, env EnvConfig) (*TweetStore, error) {
	return &TweetStore{
		logger: logger,
		db:     db,
		botID:  botID,
		env:    env,
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

	// Check if we're already participating in this conversation
	var participatingCount int64
	if tweet.ConversationID != "" {
		s.db.Table("tweets").
			Where("conversation_id = ? AND is_participating = ?",
				tweet.ConversationID, true).
			Count(&participatingCount)
	}

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

	// Prepare base tweet data
	tweetData := map[string]interface{}{
		"id":                  tweet.ID,
		"text":                tweet.Text,
		"conversation_id":     tweet.ConversationID,
		"author_id":           tweet.AuthorID,
		"author_name":         authorName,
		"author_username":     authorUsername,
		"category":            category,
		"processed_at":        now,
		"last_updated":        now,
		"conversation_ref":    conversationRef,
		"attachments":         tweet.Attachments,
		"context_annotations": tweet.ContextAnnotations,
		"edit_controls":       tweet.EditControls,
		"entities":            tweet.Entities,
		"geo":                 tweet.Geo,
		"lang":                tweet.Lang,
		"possibly_sensitive":  tweet.PossiblySensitive,
		"public_metrics":      tweet.PublicMetrics,
		"referenced_tweets":   tweet.ReferencedTweets,
		"reply_settings":      tweet.ReplySettings,
		"source":              tweet.Source,
		"withheld":            tweet.Withheld,
		"needs_reply":         true,
		"is_participating":    participatingCount > 0,
	}

	// Handle replies specifically
	if tweet.ConversationID != "" && tweet.ReferencedTweets != nil {
		for _, ref := range tweet.ReferencedTweets {
			if ref.Type == "replied_to" {
				s.logger.WithFields(logrus.Fields{
					"tweet_id":        tweet.ID,
					"parent_id":       ref.ID,
					"conversation_id": tweet.ConversationID,
				}).Debug("Processing new reply in conversation")

				// Update parent tweet
				updateResult := s.db.Table("tweets").
					Where("id = ?", ref.ID).
					Updates(map[string]interface{}{
						"unread_replies": gorm.Expr("unread_replies + 1"),
						"needs_reply":    true,
						"last_updated":   now,
					})

				if updateResult.Error != nil {
					s.logger.WithError(updateResult.Error).Error("Failed to update parent tweet")
				}

				// Update all tweets in conversation
				s.db.Table("tweets").
					Where("conversation_id = ? AND id != ?",
						tweet.ConversationID, ref.ID).
					Updates(map[string]interface{}{
						"is_participating": true,
						"last_updated":     now,
					})
				break
			}
		}
	}

	// Handle edit_history_tweet_ids as a proper array
	if tweet.EditHistoryTweetIDs != nil {
		var historyIDs pq.StringArray
		if len(tweet.EditHistoryTweetIDs) == 0 {
			historyIDs = pq.StringArray{tweet.ID}
		} else {
			historyIDs = pq.StringArray(tweet.EditHistoryTweetIDs)
		}
		tweetData["edit_history_tweet_ids"] = historyIDs
	}

	// Perform upsert operation
	result := s.db.Table("tweets").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(tweetData),
		}).
		Create(tweetData)

	if result.Error != nil {
		return fmt.Errorf("failed to save tweet: %w", result.Error)
	}

	s.logger.WithFields(logrus.Fields{
		"tweet_id":          tweet.ID,
		"conversation_id":   tweet.ConversationID,
		"category":          category,
		"public_metrics":    tweet.PublicMetrics,
		"referenced_tweets": tweet.ReferencedTweets,
		"author_name":       authorName,
		"author_username":   authorUsername,
	}).Info("Successfully saved tweet to database")

	return nil
}

// SaveAgentReply stores our own replies in the database
func (s *TweetStore) SaveAgentReply(originalTweetID, replyTweetID, conversationID string, replyText string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Save our reply with ALL required fields
	tweetData := map[string]interface{}{
		// Required fields
		"processed_at":   now,
		"process_count":  0,
		"needs_reply":    false, // Our own tweets never need replies
		"unread_replies": 0,
		"reply_count":    0,

		// Existing fields
		"id":               replyTweetID,
		"text":             replyText,
		"conversation_id":  conversationID,
		"created_at":       now,
		"category":         CategoryReply,
		"is_participating": true,
		"replied_to":       false,
		"last_updated":     now,
		"author_id":        s.botID,
		"author_name":      AgentName,
		"author_username":  AgentUsername,
		"conversation_ref": &ConversationRef{
			ConversationID: conversationID,
			ParentID:       originalTweetID,
			IsRoot:         false,
			LastReplyAt:    now,
		},
	}

	// Start a transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Save the reply tweet
		if err := tx.Table("tweets").Create(tweetData).Error; err != nil {
			return fmt.Errorf("failed to save agent reply: %w", err)
		}

		// Update the original tweet status
		if err := tx.Table("tweets").
			Where("id = ?", originalTweetID).
			Updates(map[string]interface{}{
				"replied_to":       true,
				"needs_reply":      false,
				"last_reply_id":    replyTweetID,
				"last_reply_time":  now,
				"last_updated":     now,
				"is_participating": true,
			}).Error; err != nil {
			return fmt.Errorf("failed to update original tweet: %w", err)
		}

		// Update all tweets in the conversation
		if err := tx.Table("tweets").
			Where("conversation_id = ?", conversationID).
			Updates(map[string]interface{}{
				"is_participating": true,
				"last_updated":     now,
			}).Error; err != nil {
			return fmt.Errorf("failed to update conversation tweets: %w", err)
		}

		return nil
	})
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

// UpdateTweetAfterReply updates the tweet status after we've posted a reply
func (s *TweetStore) UpdateTweetAfterReply(tweetID string, replyTweetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	result := s.db.Table("tweets").
		Where("id = ?", tweetID).
		Updates(map[string]interface{}{
			"replied_to":       true,
			"needs_reply":      false,
			"last_reply_id":    replyTweetID,
			"last_reply_time":  now,
			"last_updated":     now,
			"is_participating": true,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update tweet after reply: %w", result.Error)
	}

	s.logger.WithFields(logrus.Fields{
		"tweet_id":   tweetID,
		"reply_id":   replyTweetID,
		"updated_at": now,
	}).Debug("Updated tweet status after posting reply")

	return nil
}

// Add this method to your TweetStore struct
func (ts *TweetStore) UpdateBotID(ctx context.Context, botID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.botID = botID
	return nil
}
