package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// TweetNeedingReply represents a tweet that needs a response
type TweetNeedingReply struct {
	TweetID         string    `json:"tweet_id"`
	ConversationID  string    `json:"conversation_id"`
	LastReplyID     string    `json:"last_reply_id"`
	LastReplyTime   time.Time `json:"last_reply_time"`
	UnreadReplies   int       `json:"unread_replies"`
	IsParticipating bool      `json:"is_participating"`
	Text            string    `json:"text"`
	AuthorID        string    `json:"author_id"`
}

// RecallTweetsNeedingReply finds tweets in conversations where:
// 1. We have participated (replied)
// 2. There are new replies after our last reply
// 3. We haven't processed those replies yet
func (s *TweetStore) RecallTweetsNeedingReply(ctx context.Context, client TwitterClient) ([]TweetNeedingReply, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log := s.logger.WithField("method", "RecallTweetsNeedingReply")

	// Get authenticated user ID
	userID, err := client.GetAuthenticatedUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user ID: %w", err)
	}

	log.WithField("user_id", userID).Debug("Found authenticated user ID")

	// Debug: Log total tweets in store
	log.WithField("total_tweets", len(s.tweets)).Debug("Total tweets in store")

	// Track conversations we need to reply to
	conversationsMap := make(map[string]*TweetNeedingReply)

	// First pass: identify tweets that need our reply
	for _, tweet := range s.tweets {
		// Debug: Log each tweet being processed
		log.WithFields(logrus.Fields{
			"tweet_id":    tweet.ID,
			"author_id":   tweet.AuthorID,
			"mentions":    tweet.Entities.Mentions,
			"in_reply_to": tweet.InReplyToUserID,
		}).Debug("Processing tweet")

		// Skip our own tweets
		if tweet.AuthorID == userID {
			log.WithField("tweet_id", tweet.ID).Debug("Skipping our own tweet")
			continue
		}

		// Check if this tweet mentions us or is a reply to us
		needsReply := false
		if tweet.InReplyToUserID == userID {
			needsReply = true
			log.WithField("tweet_id", tweet.ID).Debug("Tweet is a reply to us")
		} else if tweet.Entities.Mentions != nil {
			for _, mention := range tweet.Entities.Mentions {
				if mention.ID == userID {
					needsReply = true
					log.WithField("tweet_id", tweet.ID).Debug("Tweet mentions us")
					break
				}
			}
		}

		if needsReply {
			tweetTime, err := time.Parse(time.RFC3339, tweet.CreatedAt)
			if err != nil {
				log.WithError(err).Error("Failed to parse tweet time")
				continue
			}

			log.WithFields(logrus.Fields{
				"tweet_id": tweet.ID,
				"time":     tweetTime,
			}).Debug("Found tweet needing reply")

			// Initialize or update conversation tracking
			if existing, exists := conversationsMap[tweet.ConversationID]; !exists {
				conversationsMap[tweet.ConversationID] = &TweetNeedingReply{
					TweetID:         tweet.ID,
					ConversationID:  tweet.ConversationID,
					LastReplyTime:   tweetTime,
					UnreadReplies:   1,
					IsParticipating: true,
					Text:            tweet.Text,
					AuthorID:        tweet.AuthorID,
				}
				log.WithField("conversation_id", tweet.ConversationID).Debug("Created new conversation entry")
			} else if tweetTime.After(existing.LastReplyTime) {
				// Update to the most recent tweet needing reply
				existing.TweetID = tweet.ID
				existing.LastReplyTime = tweetTime
				existing.UnreadReplies++
				existing.Text = tweet.Text
				existing.AuthorID = tweet.AuthorID
				log.WithField("conversation_id", tweet.ConversationID).Debug("Updated existing conversation")
			}
		}
	}

	// Convert map to slice for return
	var needingReply []TweetNeedingReply
	for _, reply := range conversationsMap {
		if reply.UnreadReplies > 0 {
			needingReply = append(needingReply, *reply)
		}
	}

	log.WithFields(logrus.Fields{
		"conversations_found": len(conversationsMap),
		"needs_reply":         len(needingReply),
		"conversations":       conversationsMap,
	}).Debug("Completed recall of tweets needing reply")

	return needingReply, nil
}

// TwitterClient interface defines the methods we need from the Twitter client
type TwitterClient interface {
	GetAuthenticatedUserID(ctx context.Context) (string, error)
}
