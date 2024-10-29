package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetTweetByIDParams holds the parameters for the GetTweetByID request
type GetTweetByIDParams struct {
	TweetID string
}

// GetTweetByID retrieves information about a single tweet by its ID
// Rate limit: 300/15m (app), 900/15m (user)
func (c *TwitterClient) GetTweetByID(ctx context.Context, params GetTweetByIDParams) (chan *TweetResponse, chan error) {
	dataChan := make(chan *TweetResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		log := c.logger.WithFields(logrus.Fields{
			"method":   "GetTweetByID",
			"tweet_id": params.TweetID,
		})

		endpoint := fmt.Sprintf("%s/%s", c.config.TweetEndpoint, params.TweetID)

		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Prepare query parameters
			body := map[string]interface{}{
				"tweet.fields": strings.Join(append(
					c.config.GetTweetFields(),
					"conversation_id",
					"in_reply_to_user_id",
					"referenced_tweets",
				), ","),
				"expansions": strings.Join(append(
					c.config.GetExpansions(),
					"referenced_tweets.id",
					"in_reply_to_user_id",
				), ","),
			}

			log.WithFields(logrus.Fields{
				"endpoint": endpoint,
				"params":   body,
			}).Debug("Fetching tweet")

			resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, body)
			if err != nil {
				log.WithError(err).Error("Failed to fetch tweet")
				errChan <- fmt.Errorf("failed to fetch tweet: %w", err)
				return
			}
			defer resp.Body.Close()

			var tweetResp TweetResponse
			if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
				log.WithError(err).Error("Failed to decode response")
				errChan <- fmt.Errorf("failed to decode response: %w", err)
				return
			}

			// Validate conversation_id presence
			tweet, err := tweetResp.UnmarshalTweet()
			if err != nil {
				log.WithError(err).Error("Failed to unmarshal tweet")
				errChan <- fmt.Errorf("failed to unmarshal tweet: %w", err)
				return
			}

			if tweet.ConversationID == "" {
				log.Warn("Tweet response missing conversation_id")
			}

			// Check for API errors
			if len(tweetResp.Errors) > 0 {
				for _, apiErr := range tweetResp.Errors {
					log.WithFields(logrus.Fields{
						"error_code":    apiErr.Code,
						"error_message": apiErr.Message,
					}).Error("Twitter API error")
					errChan <- &apiErr
				}
				return
			}

			// Log successful response with conversation details
			log.WithFields(logrus.Fields{
				"tweet_found":       tweetResp.Data != nil,
				"conversation_id":   tweet.ConversationID,
				"in_reply_to_user":  tweet.InReplyToUserID,
				"referenced_tweets": tweet.ReferencedTweets,
			}).Debug("Received tweet response")

			// Send the response to the data channel
			dataChan <- &tweetResp
		}
	}()

	return dataChan, errChan
}
