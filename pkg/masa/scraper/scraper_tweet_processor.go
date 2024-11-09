package scraper

import (
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/masa/masatwitter"
	"github.com/sirupsen/logrus"
)

// TweetProcessor handles the processing of retrieved tweets
type TweetProcessor struct {
	logger *logrus.Logger
}

// NewTweetProcessor creates a new TweetProcessor instance
func NewTweetProcessor(logger *logrus.Logger) *TweetProcessor {
	return &TweetProcessor{
		logger: logger,
	}
}

// ProcessTweets handles the processing of retrieved tweets, including logging
// detailed information about each tweet in the batch.
func (p *TweetProcessor) ProcessTweets(tweets []masatwitter.Tweet) {
	p.logger.WithFields(logrus.Fields{
		"tweet_count": len(tweets),
		"start_time":  time.Now().Format(time.RFC3339),
	}).Info("Processing batch of tweets")

	for _, tweet := range tweets {
		p.logger.WithFields(logrus.Fields{
			"tweet_id":        tweet.ID,
			"author_id":       tweet.UserID,
			"created_at":      tweet.TimeParsed.Format(time.RFC3339),
			"text":            tweet.Text,
			"conversation_id": tweet.ConversationID,
			"is_reply":        tweet.IsReply,
			"is_retweet":      tweet.IsRetweet,
			"is_quoted":       tweet.IsQuoted,
			"likes":           tweet.Likes,
			"retweets":        tweet.Retweets,
			"replies":         tweet.Replies,
		}).Info("Tweet details")
	}

	p.logger.WithFields(logrus.Fields{
		"tweet_count": len(tweets),
		"end_time":    time.Now().Format(time.RFC3339),
	}).Info("Completed processing batch of tweets")
}
