package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// TweetNeedingReply represents a tweet that needs a response
type TweetNeedingReply struct {
	TweetID         string          `json:"tweet_id" gorm:"primaryKey;column:id"`
	ConversationID  string          `json:"conversation_id" gorm:"column:conversation_id"`
	LastReplyID     string          `json:"last_reply_id" gorm:"column:last_reply_id"`
	LastReplyTime   time.Time       `json:"last_reply_time" gorm:"column:last_reply_time"`
	UnreadReplies   int             `json:"unread_replies" gorm:"column:unread_replies"`
	IsParticipating bool            `json:"is_participating" gorm:"column:is_participating"`
	Text            string          `json:"text" gorm:"column:text"`
	AuthorID        string          `json:"author_id" gorm:"column:author_id"`
	CreatedAt       time.Time       `json:"created_at" gorm:"column:created_at"`
	Category        string          `json:"category" gorm:"column:category"`
	AuthorName      string          `json:"author_name" gorm:"column:author_name"`
	AuthorUsername  string          `json:"author_username" gorm:"column:author_username"`
	RepliedTo       bool            `json:"replied_to" gorm:"column:replied_to"`
	ReplyCount      int             `json:"reply_count" gorm:"column:reply_count"`
	InReplyToUserID string          `json:"in_reply_to_user_id" gorm:"column:in_reply_to_user_id"`
	ConversationRef json.RawMessage `json:"conversation_ref" gorm:"column:conversation_ref"`
	Entities        json.RawMessage `json:"entities" gorm:"column:entities"`
	Lang            string          `json:"lang" gorm:"column:lang"`
}

// ConversationThread represents a complete conversation thread needing reply
type ConversationThread struct {
	ConversationID string
	Tweets         []TweetNeedingReply
	LastReplyTime  time.Time
}

// EnvConfig interface defines methods for accessing environment variables
type EnvConfig interface {
	GetString(key string) string
}

// TwitterClient interface defines the methods we need from the Twitter client
type TwitterClient interface {
	GetAuthenticatedUserID(ctx context.Context) (string, error)
}

// RecallTweetsNeedingReply finds conversations where:
// 1. We have participated (replied) or are mentioned
// 2. There are new replies after our last reply
// 3. We haven't processed those replies yet
func (s *TweetStore) RecallTweetsNeedingReply(ctx context.Context, client TwitterClient) ([]ConversationThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log := s.logger.WithField("method", "RecallTweetsNeedingReply")

	// First try to get userID from env or stored botID
	var userID string
	if envUserID := s.env.GetString("TWITTER_USER_ID"); envUserID != "" {
		userID = envUserID
		log.WithField("source", "env").Debug("Using user ID from environment")
	} else if s.botID != "" {
		userID = s.botID
		log.WithField("source", "store").Debug("Using stored bot ID")
	} else {
		// Fallback to API call if no stored ID
		var err error
		userID, err = client.GetAuthenticatedUserID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get authenticated user ID: %w", err)
		}
		log.WithField("source", "api").Debug("Retrieved user ID from API")
	}

	log.WithField("user_id", userID).Debug("Using user ID for tweet recall")

	var needingReply []TweetNeedingReply

	// Query the database for tweets needing reply
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table("tweets").
			Select(`
				tweets.id,
				tweets.text,
				tweets.conversation_id,
				tweets.created_at,
				tweets.category,
				tweets.author_id,
				tweets.author_name,
				tweets.author_username,
				tweets.last_reply_id,
				tweets.last_reply_time,
				tweets.unread_replies,
				tweets.is_participating,
				tweets.replied_to,
				tweets.reply_count,
				tweets.in_reply_to_user_id,
				tweets.conversation_ref,
				tweets.entities,
				tweets.lang
			`).
			Joins(`
				LEFT JOIN (
					SELECT 
						conversation_id,
						MAX(created_at) as bot_reply_time
					FROM tweets
					WHERE author_id = ? 
					AND category = 'reply'
					GROUP BY conversation_id
				) last_bot_reply ON tweets.conversation_id = last_bot_reply.conversation_id
			`, userID).
			Where(`
				tweets.author_id != ? AND (
					-- Case 1: New mentions needing initial reply
					(tweets.category = 'mention' AND tweets.replied_to = FALSE)
					OR
					-- Case 2: New conversation tweets needing response
					(tweets.category = 'conversation' AND tweets.replied_to = FALSE)
					OR
					-- Case 3: Active conversations with new activity
					(
						tweets.conversation_id IN (
							SELECT DISTINCT conversation_id 
							FROM tweets 
							WHERE is_participating = TRUE
						)
						AND tweets.replied_to = FALSE
						AND (
							tweets.unread_replies > 0
							OR
							tweets.created_at > COALESCE(last_bot_reply.bot_reply_time, '1970-01-01')
						)
					)
				)
			`, userID).
			Order("tweets.created_at ASC")

		// Add debug logging for the query
		log.WithFields(logrus.Fields{
			"sql":  query.Statement.SQL.String(),
			"vars": query.Statement.Vars,
		}).Debug("Executing recall query")

		result := query.Find(&needingReply)
		if result.Error != nil {
			return fmt.Errorf("failed to query tweets needing reply: %w", result.Error)
		}

		log.WithFields(logrus.Fields{
			"tweets_found": len(needingReply),
			"query_time":   time.Now(),
		}).Debug("Found tweets needing reply")

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("database transaction failed: %w", err)
	}

	// Group tweets by conversation with debug logging
	conversationsMap := make(map[string]*ConversationThread)
	for i := range needingReply {
		tweet := needingReply[i]

		thread, exists := conversationsMap[tweet.ConversationID]
		if !exists {
			// Get full conversation context
			var contextTweets []TweetNeedingReply
			err := s.db.Table("tweets").
				Where("conversation_id = ?", tweet.ConversationID).
				Order("created_at ASC").
				Find(&contextTweets).Error

			if err != nil {
				log.WithError(err).Error("Failed to get conversation context")
				continue
			}

			thread = &ConversationThread{
				ConversationID: tweet.ConversationID,
				Tweets:         contextTweets,
				LastReplyTime:  tweet.LastReplyTime,
			}
			conversationsMap[tweet.ConversationID] = thread
		}

		// Update thread metadata
		if tweet.LastReplyTime.After(thread.LastReplyTime) {
			thread.LastReplyTime = tweet.LastReplyTime
		}

		log.WithFields(logrus.Fields{
			"tweet_id":         tweet.TweetID,
			"conversation_id":  tweet.ConversationID,
			"context_tweets":   len(thread.Tweets),
			"is_participating": tweet.IsParticipating,
			"category":         tweet.Category,
		}).Debug("Processing tweet with context")
	}

	// Convert map to slice and ensure tweets are properly ordered
	var result []ConversationThread
	for _, thread := range conversationsMap {
		// Sort tweets within each thread by creation time
		sort.Slice(thread.Tweets, func(i, j int) bool {
			return thread.Tweets[i].CreatedAt.Before(thread.Tweets[j].CreatedAt)
		})

		// Check if thread needs reply
		needsReply := false
		for _, tweet := range thread.Tweets {
			if tweet.AuthorID != userID && (tweet.UnreadReplies > 0 ||
				(tweet.Category == "mention" && !tweet.RepliedTo) ||
				(tweet.Category == "conversation" && !tweet.RepliedTo) ||
				(tweet.IsParticipating && tweet.CreatedAt.After(thread.LastReplyTime))) {
				needsReply = true
				break
			}
		}

		log.WithFields(logrus.Fields{
			"conversation_id": thread.ConversationID,
			"tweets_count":    len(thread.Tweets),
			"needs_reply":     needsReply,
			"last_reply_time": thread.LastReplyTime,
		}).Debug("Processing conversation thread")

		if needsReply {
			result = append(result, *thread)
		}
	}

	log.WithFields(logrus.Fields{
		"conversations_found": len(result),
		"total_tweets":        len(needingReply),
		"query_time":          time.Now(),
	}).Info("Completed recall of conversation threads needing reply")

	return result, nil
}
