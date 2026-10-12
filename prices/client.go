// Package prices queries the OSRS Wiki Real-time Prices API.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices
package prices

import (
	"net/http"
	"time"

	"github.com/justintout/go-osrs/cache"
	"github.com/justintout/go-osrs/internal/transport"
)

const (
	defaultBaseURL   = "https://prices.runescape.wiki/api/v1/osrs"
	deadmanRebornURL = "https://prices.runescape.wiki/api/v1/dmm"
	freshStartURL    = "https://prices.runescape.wiki/api/v1/fsw"
)

// Client queries the OSRS Wiki Real-time Prices API.
type Client struct {
	t       *transport.Client
	baseURL string
}

// NewClient creates a Client. The Wiki team requires a descriptive user
// agent, such as "my-flip-bot - @username on Discord".
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Please_set_a_descriptive_User-Agent!
func NewClient(userAgent string, options ...Option) *Client {
	c := &Client{
		t:       transport.New(http.DefaultClient, userAgent, 100*time.Millisecond),
		baseURL: defaultBaseURL,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

// Option configures a Client.
type Option func(*Client)

// WithCache stores responses in c, following the server's freshness
// headers and revalidating with ETags where the server sends them.
func WithCache(c *cache.Cache) Option {
	return func(cl *Client) { cl.t.Cache = c }
}

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.t.HTTP = httpClient }
}

// ForDeadmanReborn queries Deadman Reborn prices.
func ForDeadmanReborn() Option {
	return func(c *Client) { c.baseURL = deadmanRebornURL }
}

// ForFreshStart queries Fresh Start World prices.
func ForFreshStart() Option {
	return func(c *Client) { c.baseURL = freshStartURL }
}

// WithBaseURL sets the API base URL, for proxies and tests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}
