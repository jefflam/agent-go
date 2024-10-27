package twitter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mrjones/oauth"
	"github.com/sirupsen/logrus"
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
	log := logrus.WithFields(logrus.Fields{
		"component": "Authenticator",
		"method":    "NewAuthenticator",
	})

	log.Debug("Creating new authenticator with config", "config", config)

	// For write operations (POST tweets), we need OAuth 1.0a
	if config.ConsumerKey != "" && config.AccessToken != "" {
		log.Debug("Using OAuth 1.0a authentication")
		return newUserAuthenticator(
			config.ConsumerKey,
			config.ConsumerSecret,
			config.AccessToken,
			config.AccessTokenSecret,
		)
	}

	// For read-only operations, we can use Bearer token
	if config.BearerToken != "" {
		log.Debug("Using Bearer token authentication")
		return newAppAuthenticator(config.BearerToken)
	}

	log.Error("No valid authentication credentials provided")
	return nil, fmt.Errorf("either OAuth 1.0a credentials or Bearer token must be provided")
}

func newAppAuthenticator(bearerToken string) (*Authenticator, error) {
	log := logrus.WithFields(logrus.Fields{
		"component": "Authenticator",
		"method":    "newAppAuthenticator",
	})

	log.Debug("Creating new app authenticator")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	auth := &Authenticator{
		client:      client,
		bearerToken: bearerToken,
	}

	log.Debug("App authenticator created successfully")
	return auth, nil
}

func newUserAuthenticator(consumerKey, consumerSecret, accessToken, accessTokenSecret string) (*Authenticator, error) {
	log := logrus.WithFields(logrus.Fields{
		"component": "Authenticator",
		"method":    "newUserAuthenticator",
	})

	log.Debug("Creating new user authenticator")

	consumer := oauth.NewConsumer(consumerKey, consumerSecret, oauth.ServiceProvider{
		RequestTokenUrl:   RequestTokenURL,
		AuthorizeTokenUrl: AuthorizeTokenURL,
		AccessTokenUrl:    AccessTokenURL,
	})

	consumer.HttpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	token := oauth.AccessToken{
		Token:  accessToken,
		Secret: accessTokenSecret,
	}

	client, err := consumer.MakeHttpClient(&token)
	if err != nil {
		log.WithError(err).Error("Failed to create OAuth client")
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	auth := &Authenticator{
		client:            client,
		consumerKey:       consumerKey,
		consumerSecret:    consumerSecret,
		accessToken:       accessToken,
		accessTokenSecret: accessTokenSecret,
	}

	log.Debug("User authenticator created successfully")
	return auth, nil
}

func (a *Authenticator) GetClient() *http.Client {
	return a.client
}

func (a *Authenticator) SetAuthHeader(req *http.Request) error {
	log := logrus.WithFields(logrus.Fields{
		"component": "Authenticator",
		"method":    "SetAuthHeader",
		"url":       req.URL.String(),
	})

	log.Debug("Setting authentication header")

	if a.bearerToken != "" {
		log.Debug("Using Bearer token authentication")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.bearerToken))
		return nil
	}

	log.Debug("Using OAuth 1.0a authentication")
	return nil
}
