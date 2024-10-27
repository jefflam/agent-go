package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetTweetsParams holds the parameters for the GetTweets request
type GetTweetsParams struct {
	TweetIDs        []string // List of tweet IDs to fetch
	PaginationToken string
	MaxResults      int
}

// GetTweets retrieves information about specific tweets by their IDs
// Rate limit: 300/15m (app), 900/15m (user)
func (c *TwitterClient) GetTweets(ctx context.Context, params GetTweetsParams) (chan *TweetResponse, chan error) {
	dataChan := make(chan *TweetResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		log := c.logger.WithFields(logrus.Fields{
			"method":     "GetTweets",
			"tweet_ids":  params.TweetIDs,
			"num_tweets": len(params.TweetIDs),
		})

		endpoint := c.config.TweetEndpoint

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Prepare query parameters
				body := map[string]interface{}{
					"ids":              strings.Join(params.TweetIDs, ","),
					"pagination_token": params.PaginationToken,
					"max_results":      params.MaxResults,
					"tweet.fields": strings.Join(append(
						c.config.GetTweetFields(),
						"conversation_id",
					), ","),
					"expansions": c.config.GetExpansions(),
				}

				log.WithFields(logrus.Fields{
					"endpoint": endpoint,
					"params":   body,
				}).Debug("Fetching tweets")

				resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, body)
				if err != nil {
					log.WithError(err).Error("Failed to fetch tweets")
					errChan <- fmt.Errorf("failed to fetch tweets: %w", err)
					return
				}
				defer resp.Body.Close()

				var tweetResp TweetResponse
				if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
					log.WithError(err).Error("Failed to decode response")
					errChan <- fmt.Errorf("failed to decode response: %w", err)
					return
				}

				// Log the response details
				log.WithFields(logrus.Fields{
					"tweet_received": tweetResp.Data != nil,
					"result_count":   tweetResp.Meta.ResultCount,
				}).Debug("Received tweets response")

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
