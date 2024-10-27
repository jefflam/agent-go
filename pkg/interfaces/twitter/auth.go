package twitter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mrjones/oauth"
)

const (
	BaseURL           = "https://api.twitter.com/2"
	RequestTokenURL   = "https://api.twitter.com/oauth/request_token"
	AuthorizeTokenURL = "https://api.twitter.com/oauth/authorize"
	AccessTokenURL    = "https://api.twitter.com/oauth/access_token"
	TokenURL          = "https://api.twitter.com/oauth2/token"
)

type Authenticator struct {
	client            *http.Client
	consumerKey       string
	consumerSecret    string
	accessToken       string
	accessTokenSecret string
	bearerToken       string
}

func NewAuthenticator(config *TwitterConfig) (*Authenticator, error) {
	// For write operations (POST tweets), we need OAuth 1.0a
	if config.ConsumerKey != "" && config.AccessToken != "" {
		return newUserAuthenticator(
			config.ConsumerKey,
			config.ConsumerSecret,
			config.AccessToken,
			config.AccessTokenSecret,
		)
	}

	// For read-only operations, we can use Bearer token
	if config.BearerToken != "" {
		return newAppAuthenticator(config.BearerToken)
	}

	return nil, fmt.Errorf("either OAuth 1.0a credentials or Bearer token must be provided")
}

func newAppAuthenticator(bearerToken string) (*Authenticator, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Authenticator{
		client:      client,
		bearerToken: bearerToken,
	}, nil
}

func newUserAuthenticator(consumerKey, consumerSecret, accessToken, accessTokenSecret string) (*Authenticator, error) {
	consumer := oauth.NewConsumer(consumerKey, consumerSecret, oauth.ServiceProvider{
		RequestTokenUrl:   RequestTokenURL,
		AuthorizeTokenUrl: AuthorizeTokenURL,
		AccessTokenUrl:    AccessTokenURL,
	})

	// Add timeout configuration
	consumer.HttpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	token := oauth.AccessToken{
		Token:  accessToken,
		Secret: accessTokenSecret,
	}

	client, err := consumer.MakeHttpClient(&token)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	return &Authenticator{
		client:            client,
		consumerKey:       consumerKey,
		consumerSecret:    consumerSecret,
		accessToken:       accessToken,
		accessTokenSecret: accessTokenSecret,
	}, nil
}

func (a *Authenticator) GetClient() *http.Client {
	return a.client
}

func (a *Authenticator) SetAuthHeader(req *http.Request) error {
	if a.bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.bearerToken))
		return nil
	}

	// OAuth 1.0a client already handles authentication
	return nil
}
