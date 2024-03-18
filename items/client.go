package items

import (
	"encoding/json"
	"net/http"
)

const (
	defaultBaseURL   = "prices.runescape.wiki/api/v1/osrs"
	deadmanRebornURL = "prices.runescape.wiki/api/v1/dmm"
	freshStartURL    = "prices.runescape.wiki/api/v1/fsw"
)

const defaultUserAgent = "go-osrs v0.0.1 https://github.com/justintout/go-osrs"

type Client struct {
	httpClient *http.Client
	baseUrl    string
	userAgent  string
}

func NewClient(options ...ClientOption) *Client {
	client := &Client{
		httpClient: http.DefaultClient,
		baseUrl:    defaultBaseURL,
		userAgent:  defaultUserAgent,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

// Latest returns the latest price for all items
func (c *Client) Latest() (Prices, error) {
	res, err := c.httpClient.Get("https://" + c.baseUrl + "/latest")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var pr priceResponse
	err = json.NewDecoder(res.Body).Decode(&pr)
	if err != nil {
		return nil, err
	}
	return pr.Data, nil
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

func ForDeadmanReborn() ClientOption {
	return func(c *Client) {
		c.baseUrl = deadmanRebornURL
	}
}

func ForFreshStart() ClientOption {
	return func(c *Client) {
		c.baseUrl = freshStartURL
	}
}
