// Package wom queries the Wise Old Man API: tracked players, gains,
// snapshots, achievements, groups, and competitions.
// https://docs.wiseoldman.net
//
// Without an API key, Wise Old Man allows 20 requests a minute. Over the
// limit, calls return an error matching osrs.ErrRateLimited.
package wom

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/justintout/go-osrs/internal/transport"
)

const defaultBaseURL = "https://api.wiseoldman.net/v2"

// Client queries the Wise Old Man API.
type Client struct {
	t       transport.Client
	baseURL string
}

// NewClient creates a Client. Wise Old Man asks for a descriptive user
// agent.
func NewClient(userAgent string, options ...Option) *Client {
	c := &Client{
		t:       transport.Client{HTTP: http.DefaultClient, UserAgent: userAgent, Header: http.Header{}},
		baseURL: defaultBaseURL,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.t.HTTP = httpClient }
}

// WithAPIKey sends a Wise Old Man API key, which raises the rate limit.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.t.Header.Set("x-api-key", key) }
}

// WithBaseURL sets the API base URL, for proxies and tests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// Period is a preset time range for gains and snapshots.
type Period string

const (
	FiveMinutes Period = "five_min"
	Day         Period = "day"
	Week        Period = "week"
	Month       Period = "month"
	Year        Period = "year"
)

// get fetches path and decodes it as T, returning the zero T on error.
func get[T any](ctx context.Context, c *Client, path string, query url.Values) (T, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var v T
	if err := c.t.GetJSON(ctx, u, &v); err != nil {
		var zero T
		return zero, err
	}
	return v, nil
}

func playerPath(username string, rest ...string) string {
	p := "/players/" + url.PathEscape(username)
	for _, r := range rest {
		p += "/" + r
	}
	return p
}

func idPath(kind string, id int, rest string) string {
	return fmt.Sprintf("/%s/%d%s", kind, id, rest)
}
