package models

import (
	"time"
)

// TweetCategory represents the type of tweet interaction
type TweetCategory string

const (
	CategoryMention      TweetCategory = "mention"
	CategoryReply        TweetCategory = "reply"
	CategoryQuote        TweetCategory = "quote"
	CategoryRetweet      TweetCategory = "retweet"
	CategoryDM           TweetCategory = "dm"
	CategoryConversation TweetCategory = "conversation"
)

// Tweet represents the database model for tweets
type Tweet struct {
	ID             string    `gorm:"primaryKey;column:id"`
	Text           string    `gorm:"column:text;not null"`
	ConversationID string    `gorm:"column:conversation_id"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`

	// Author Information
	AuthorID       string `gorm:"column:author_id;not null"`
	AuthorName     string `gorm:"column:author_name"`
	AuthorUsername string `gorm:"column:author_username"`

	// Operational Fields
	Category        TweetCategory `gorm:"column:category;type:tweet_category;not null"`
	ProcessedAt     time.Time     `gorm:"column:processed_at;not null"`
	LastUpdated     time.Time     `gorm:"column:last_updated;not null"`
	ProcessCount    int           `gorm:"column:process_count;default:0"`
	ConversationRef interface{}   `gorm:"column:conversation_ref;type:jsonb"`
	NeedsReply      bool          `gorm:"column:needs_reply;default:true"`
	IsParticipating bool          `gorm:"column:is_participating;default:false"`

	// Reply Tracking
	RepliedTo       bool      `gorm:"column:replied_to;default:false"`
	LastReplyID     string    `gorm:"column:last_reply_id"`
	LastReplyTime   time.Time `gorm:"column:last_reply_time"`
	ReplyCount      int       `gorm:"column:reply_count;default:0"`
	UnreadReplies   int       `gorm:"column:unread_replies;default:0"`
	InReplyToUserID string    `gorm:"column:in_reply_to_user_id"`

	// Twitter API Response Fields
	Attachments         interface{} `gorm:"column:attachments;type:jsonb"`
	ContextAnnotations  interface{} `gorm:"column:context_annotations;type:jsonb"`
	EditControls        interface{} `gorm:"column:edit_controls;type:jsonb"`
	EditHistoryTweetIDs []string    `gorm:"column:edit_history_tweet_ids;type:text[]"`
	Entities            interface{} `gorm:"column:entities;type:jsonb"`
	Geo                 interface{} `gorm:"column:geo;type:jsonb"`
	Lang                string      `gorm:"column:lang"`
	NonPublicMetrics    interface{} `gorm:"column:non_public_metrics;type:jsonb"`
	OrganicMetrics      interface{} `gorm:"column:organic_metrics;type:jsonb"`
	PossiblySensitive   bool        `gorm:"column:possibly_sensitive"`
	PromotedMetrics     interface{} `gorm:"column:promoted_metrics;type:jsonb"`
	PublicMetrics       interface{} `gorm:"column:public_metrics;type:jsonb"`
	ReferencedTweets    interface{} `gorm:"column:referenced_tweets;type:jsonb"`
	ReplySettings       string      `gorm:"column:reply_settings"`
	Source              string      `gorm:"column:source"`
	Withheld            interface{} `gorm:"column:withheld;type:jsonb"`
}

// TableName specifies the table name for the Tweet model
func (Tweet) TableName() string {
	return "tweets"
}
