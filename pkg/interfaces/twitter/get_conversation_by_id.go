package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetConversationParams holds the parameters for retrieving a conversation thread
type GetConversationParams struct {
	ConversationID  string
	PaginationToken string
	MaxResults      int
}

// GetConversation retrieves all tweets in a conversation thread by conversation_id
// Rate limit: 450/15m (app), 180/15m (user)
func (c *TwitterClient) GetConversation(ctx context.Context, params GetConversationParams) (chan *ConversationResponse, chan error) {
	dataChan := make(chan *ConversationResponse)
	errChan := make(chan error)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		log := c.logger.WithFields(logrus.Fields{
			"method":          "GetConversation",
			"conversation_id": params.ConversationID,
		})

		endpoint := fmt.Sprintf("%s/search/recent", c.config.SearchEndpoint)

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Prepare query parameters with conversation_id filter
				query := fmt.Sprintf("conversation_id:%s", params.ConversationID)
				body := map[string]interface{}{
					"query":            query,
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
					"query":    query,
					"params":   body,
				}).Debug("Fetching conversation tweets")

				resp, err := c.makeRequest(ctx, http.MethodGet, endpoint, body)
				if err != nil {
					log.WithError(err).Error("Failed to fetch conversation")
					errChan <- fmt.Errorf("failed to fetch conversation: %w", err)
					return
				}
				defer resp.Body.Close()

				var conversationResp ConversationResponse
				if err := json.NewDecoder(resp.Body).Decode(&conversationResp); err != nil {
					log.WithError(err).Error("Failed to decode response")
					errChan <- fmt.Errorf("failed to decode response: %w", err)
					return
				}

				if len(conversationResp.Errors) > 0 {
					for _, apiErr := range conversationResp.Errors {
						log.WithFields(logrus.Fields{
							"error_code":    apiErr.Code,
							"error_message": apiErr.Message,
						}).Error("Twitter API error")
						errChan <- &apiErr
					}
					return
				}

				// Log successful response
				log.WithFields(logrus.Fields{
					"tweets_found": len(conversationResp.Data),
					"meta":         conversationResp.Meta,
				}).Debug("Received conversation response")

				// Send the response to the data channel
				dataChan <- &conversationResp

				// Check if we have more pages
				if conversationResp.Meta.NextToken == "" {
					log.Debug("No more pages to fetch")
					return
				}

				// Update pagination token for next request
				params.PaginationToken = conversationResp.Meta.NextToken
				log.WithField("next_token", params.PaginationToken).Debug("Fetching next page")
			}
		}
	}()

	return dataChan, errChan
}
