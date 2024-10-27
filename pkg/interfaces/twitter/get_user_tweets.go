package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetUserTweetsParams holds the parameters for the GetUserTweets request
type GetUserTweetsParams struct {
	UserID          string
	PaginationToken string
	MaxResults      int
}

// GetUserTweets retrieves tweets posted by a specific user
// Rate limit: 1500/15m (app), 900/15m (user)
func (c *TwitterClient) GetUserTweets(ctx context.Context, params GetUserTweetsParams) (chan *TweetResponse, chan error) {
	dataChan := make(chan *TweetResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		log := c.logger.WithFields(logrus.Fields{
			"method": "GetUserTweets",
			"userID": params.UserID,
		})

		endpoint := fmt.Sprintf("%s/%s/%s/tweets",
			c.config.BaseURL,
			c.config.UserEndpoint,
			params.UserID,
		)

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Prepare query parameters
				body := map[string]interface{}{
					"pagination_token": params.PaginationToken,
					"max_results":      params.MaxResults,
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
				}).Debug("Fetching user tweets")

				resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, body)
				if err != nil {
					log.WithError(err).Error("Failed to fetch user tweets")
					errChan <- fmt.Errorf("failed to fetch user tweets: %w", err)
					return
				}
				defer resp.Body.Close()

				var tweetResp TweetResponse
				if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
					log.WithError(err).Error("Failed to decode response")
					errChan <- fmt.Errorf("failed to decode response: %w", err)
					return
				}

				// Send the response to the data channel
				dataChan <- &tweetResp

				// Check if we have more pages
				if tweetResp.Meta.NextToken == "" {
					log.Debug("No more pages to fetch")
					return
				}

				// Update pagination token for next request
				params.PaginationToken = tweetResp.Meta.NextToken
				log.WithField("next_token", params.PaginationToken).Debug("Fetching next page")
			}
		}
	}()

	return dataChan, errChan
}
