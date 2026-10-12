// Package hiscores looks up players and leaderboards on the Jagex OSRS
// hiscores.
package hiscores

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/justintout/go-osrs/internal/transport"
)

const defaultBaseURL = "https://secure.runescape.com"

// Mode is a hiscores game mode. Each mode has its own leaderboard.
type Mode string

const (
	Regular    Mode = "regular"
	Ironman    Mode = "ironman"
	Hardcore   Mode = "hardcore"
	Ultimate   Mode = "ultimate"
	Deadman    Mode = "deadman"
	Seasonal   Mode = "seasonal"
	Tournament Mode = "tournament"
	FreshStart Mode = "fresh_start"
	OneDefence Mode = "1_defence"
	Skiller    Mode = "skiller"
)

var modePaths = map[Mode]string{
	Regular:    "hiscore_oldschool",
	Ironman:    "hiscore_oldschool_ironman",
	Hardcore:   "hiscore_oldschool_hardcore_ironman",
	Ultimate:   "hiscore_oldschool_ultimate",
	Deadman:    "hiscore_oldschool_deadman",
	Seasonal:   "hiscore_oldschool_seasonal",
	Tournament: "hiscore_oldschool_tournament",
	FreshStart: "hiscore_oldschool_fresh_start",
	OneDefence: "hiscore_oldschool_skiller_defence",
	Skiller:    "hiscore_oldschool_skiller",
}

// Modes lists every supported mode.
var Modes = []Mode{Regular, Ironman, Hardcore, Ultimate, Deadman, Seasonal, Tournament, FreshStart, OneDefence, Skiller}

// Client queries the Jagex hiscores.
type Client struct {
	t       *transport.Client
	baseURL string
}

// NewClient creates a Client. Jagex expects a descriptive user agent.
func NewClient(userAgent string, options ...Option) *Client {
	c := &Client{
		t:       transport.New(http.DefaultClient, userAgent, 250*time.Millisecond),
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

// WithBaseURL sets the hiscores base URL, for proxies and tests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

func (c *Client) modeURL(mode Mode, page string, query url.Values) (string, error) {
	path, ok := modePaths[mode]
	if !ok {
		return "", fmt.Errorf("hiscores: unknown mode %q", mode)
	}
	return fmt.Sprintf("%s/m=%s/%s?%s", c.baseURL, path, page, query.Encode()), nil
}
