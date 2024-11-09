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
	TweetID         string          `json:"tweet_id" gorm:"column:id"`
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

// RecallTweetsNeedingReply finds conversations where:
// 1. We have participated (replied) or are mentioned
// 2. There are new replies after our last reply
// 3. We haven't processed those replies yet
func (s *TweetStore) RecallTweetsNeedingReply(ctx context.Context, client TwitterClient) ([]ConversationThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log := s.logger.WithField("method", "RecallTweetsNeedingReply")

	// Get authenticated user ID
	userID, err := client.GetAuthenticatedUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user ID: %w", err)
	}

	log.WithField("user_id", userID).Debug("Found authenticated user ID")

	var needingReply []TweetNeedingReply

	// Query the database for tweets needing reply
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table("tweets").
			Select(`
				id as tweet_id,
				text,
				conversation_id,
				created_at,
				category,
				author_id,
				author_name,
				author_username,
				last_reply_id,
				last_reply_time,
				unread_replies,
				is_participating,
				replied_to,
				reply_count,
				in_reply_to_user_id,
				conversation_ref,
				entities,
				lang
			`).
			Where(`
				needs_reply = TRUE 
				AND author_id != ?
				AND (
					-- Case 1: Tweet hasn't been replied to yet
					(replied_to = FALSE AND needs_reply = TRUE)
					OR
					-- Case 2: New replies after our last response
					(is_participating = TRUE AND unread_replies > 0)
						OR
					-- Case 3: Part of an ongoing conversation
					(conversation_id IN (
						SELECT conversation_id 
						FROM tweets 
						WHERE is_participating = TRUE
					) AND created_at > COALESCE(last_reply_time, '1970-01-01'))
				)
			`, userID).
			Order("created_at ASC") // Order by creation time to maintain conversation flow

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

	// Group tweets by conversation
	conversationsMap := make(map[string]*ConversationThread)
	for i := range needingReply {
		tweet := needingReply[i]

		if thread, exists := conversationsMap[tweet.ConversationID]; !exists {
			conversationsMap[tweet.ConversationID] = &ConversationThread{
				ConversationID: tweet.ConversationID,
				Tweets:         []TweetNeedingReply{tweet},
				LastReplyTime:  tweet.LastReplyTime,
			}
		} else {
			thread.Tweets = append(thread.Tweets, tweet)
			if tweet.LastReplyTime.After(thread.LastReplyTime) {
				thread.LastReplyTime = tweet.LastReplyTime
			}
		}

		log.WithFields(logrus.Fields{
			"tweet_id":        tweet.TweetID,
			"conversation_id": tweet.ConversationID,
			"author_id":       tweet.AuthorID,
			"created_at":      tweet.CreatedAt,
		}).Debug("Processing tweet in conversation thread")
	}

	// Convert map to slice and ensure tweets are properly ordered
	var result []ConversationThread
	for _, thread := range conversationsMap {
		// Sort tweets within each thread by creation time
		sort.Slice(thread.Tweets, func(i, j int) bool {
			return thread.Tweets[i].CreatedAt.Before(thread.Tweets[j].CreatedAt)
		})

		// Only include threads with unread replies
		hasUnread := false
		for _, tweet := range thread.Tweets {
			if tweet.UnreadReplies > 0 {
				hasUnread = true
				break
			}
		}
		if hasUnread {
			result = append(result, *thread)
		}
	}

	// Sort threads by their most recent activity
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastReplyTime.After(result[j].LastReplyTime)
	})

	log.WithFields(logrus.Fields{
		"conversations_found": len(result),
		"total_tweets":        len(needingReply),
	}).Debug("Completed recall of conversation threads needing reply")

	return result, nil
}

// TwitterClient interface defines the methods we need from the Twitter client
type TwitterClient interface {
	GetAuthenticatedUserID(ctx context.Context) (string, error)
}
