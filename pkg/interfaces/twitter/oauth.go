package twitter

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// OAuth1 constants
const (
	oauthVersion         = "1.0"
	oauthSignatureMethod = "HMAC-SHA1"
)

type oauthParams struct {
	consumerKey     string
	consumerSecret  string
	accessToken     string
	accessSecret    string
	nonce           string
	timestamp       string
	signatureMethod string
	version         string
}

func (c *TwitterClient) generateOAuth1Header(method, urlStr string, params map[string]string) (string, error) {
	oauth := &oauthParams{
		consumerKey:     c.config.ConsumerKey,
		consumerSecret:  c.config.ConsumerSecret,
		accessToken:     c.config.AccessToken,
		accessSecret:    c.config.AccessTokenSecret,
		nonce:           generateNonce(),
		timestamp:       fmt.Sprintf("%d", time.Now().Unix()),
		signatureMethod: oauthSignatureMethod,
		version:         oauthVersion,
	}

	// Collect parameters
	allParams := make(map[string]string)
	for k, v := range params {
		allParams[k] = v
	}
	allParams["oauth_consumer_key"] = oauth.consumerKey
	allParams["oauth_nonce"] = oauth.nonce
	allParams["oauth_signature_method"] = oauth.signatureMethod
	allParams["oauth_timestamp"] = oauth.timestamp
	allParams["oauth_token"] = oauth.accessToken
	allParams["oauth_version"] = oauth.version

	// Generate signature
	signature := generateSignature(method, urlStr, allParams, oauth.consumerSecret, oauth.accessSecret)
	allParams["oauth_signature"] = signature

	// Build Authorization header
	var headerParams []string
	for k, v := range allParams {
		if strings.HasPrefix(k, "oauth_") {
			headerParams = append(headerParams, fmt.Sprintf("%s=\"%s\"", k, escape(v)))
		}
	}
	sort.Strings(headerParams)

	return "OAuth " + strings.Join(headerParams, ", "), nil
}

func generateSignature(method, urlStr string, params map[string]string, consumerSecret, tokenSecret string) string {
	// Create parameter string
	var paramPairs []string
	for k, v := range params {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", escape(k), escape(v)))
	}
	sort.Strings(paramPairs)
	paramString := strings.Join(paramPairs, "&")

	// Create signature base string
	baseURL, _ := url.Parse(urlStr)
	baseURL.RawQuery = ""
	signatureBase := strings.Join([]string{
		method,
		escape(baseURL.String()),
		escape(paramString),
	}, "&")

	// Generate signing key
	signingKey := fmt.Sprintf("%s&%s", escape(consumerSecret), escape(tokenSecret))

	// Calculate HMAC-SHA1
	h := hmac.New(sha1.New, []byte(signingKey))
	h.Write([]byte(signatureBase))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

func generateNonce() string {
	nonce := make([]byte, 32)
	for i := 0; i < len(nonce); i++ {
		nonce[i] = byte(time.Now().UnixNano() & 0xff)
	}
	return base64.StdEncoding.EncodeToString(nonce)
}

func escape(s string) string {
	// Implementation of RFC 3986 percent encoding
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}
