// Package account is the HTTP client for 9itchi-account-api. Base URL comes from env, never hard-coded.
package account

import (
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 50 * time.Second},
	}
}
