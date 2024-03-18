package items

import (
	"net/http"
)

const (
	defaultBaseURL   = "prices.runescape.wiki/api/v1/osrs"
	deadmanRebornURL = "prices.runescape.wiki/api/v1/dmm"
	freshStartURL    = "prices.runescape.wiki/api/v1/fsw"
)

// Client is used to query the OSRS Wiki Real-time Prices API
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// NewClient creates a new Client
// A descriptive user agent must be specified as requested by the Wiki team
// See https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Please_set_a_descriptive_User-Agent!
func NewClient(userAgent string, options ...ClientOption) *Client {
	client := &Client{
		httpClient: http.DefaultClient,
		baseURL:    defaultBaseURL,
		userAgent:  userAgent,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

type ClientOption func(*Client)

// WithHTTPClient is a functional option to pass an HTTP client different from http.DefaultClient
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// ForDeadmanReborn is a functional option to use the Deadman Reborn URL instead of the base game url
func ForDeadmanReborn() ClientOption {
	return func(c *Client) {
		c.baseURL = deadmanRebornURL
	}
}

// ForFreshStart is a functional option to use the Fresh Start World URL instead of the base game url
func ForFreshStart() ClientOption {
	return func(c *Client) {
		c.baseURL = freshStartURL
	}
}
