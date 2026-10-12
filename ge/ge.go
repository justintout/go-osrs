// Package ge queries the official Jagex Grand Exchange catalogue: guide
// prices, trends, and 180 days of daily price history.
//
// Guide prices update about once a day. For live trade prices, use the
// prices package.
package ge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justintout/go-osrs/internal/transport"
)

const defaultBaseURL = "https://secure.runescape.com/m=itemdb_oldschool/api"

// Client queries the Jagex Grand Exchange catalogue.
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

// WithBaseURL sets the catalogue base URL, for proxies and tests.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// Item is an item's catalogue entry.
type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Members     bool   `json:"members"`
	Icon        string `json:"icon"`
	IconLarge   string `json:"icon_large"`
	// Price and TodayChange are rounded the way Jagex displays them, such
	// as 800.4k. Use History for exact guide prices.
	Price       int `json:"price"`
	TodayChange int `json:"today_change"`
	// Change30d, Change90d, and Change180d are percent changes.
	Change30d  float64 `json:"change_30d"`
	Change90d  float64 `json:"change_90d"`
	Change180d float64 `json:"change_180d"`
}

// Item returns an item's catalogue entry. It returns an error matching
// osrs.ErrNotFound for untradeable or unknown items.
func (c *Client) Item(ctx context.Context, id int) (*Item, error) {
	var res struct {
		Item struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Members     string `json:"members"`
			Icon        string `json:"icon"`
			IconLarge   string `json:"icon_large"`
			Current     struct {
				Price json.RawMessage `json:"price"`
			} `json:"current"`
			Today struct {
				Price json.RawMessage `json:"price"`
			} `json:"today"`
			Day30  change `json:"day30"`
			Day90  change `json:"day90"`
			Day180 change `json:"day180"`
		} `json:"item"`
	}
	if err := c.t.GetJSON(ctx, fmt.Sprintf("%s/catalogue/detail.json?item=%d", c.baseURL, id), &res); err != nil {
		return nil, err
	}
	r := res.Item
	price, err := parseShortPrice(r.Current.Price)
	if err != nil {
		return nil, err
	}
	today, err := parseShortPrice(r.Today.Price)
	if err != nil {
		return nil, err
	}
	return &Item{
		ID: r.ID, Name: r.Name, Description: r.Description, Members: r.Members == "true",
		Icon: r.Icon, IconLarge: r.IconLarge, Price: price, TodayChange: today,
		Change30d: float64(r.Day30), Change90d: float64(r.Day90), Change180d: float64(r.Day180),
	}, nil
}

// Point is one day of guide price history.
type Point struct {
	Time time.Time `json:"time"`
	// Price is the exact guide price that day.
	Price int `json:"price"`
	// Average is Jagex's 30-day moving average.
	Average int `json:"average"`
}

// History returns the last 180 days of guide prices, oldest first.
func (c *Client) History(ctx context.Context, id int) ([]Point, error) {
	var res struct {
		Daily   map[int64]int `json:"daily"`
		Average map[int64]int `json:"average"`
	}
	if err := c.t.GetJSON(ctx, fmt.Sprintf("%s/graph/%d.json", c.baseURL, id), &res); err != nil {
		return nil, err
	}
	points := make([]Point, 0, len(res.Daily))
	for ms, price := range res.Daily {
		points = append(points, Point{Time: time.UnixMilli(ms), Price: price, Average: res.Average[ms]})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Time.Before(points[j].Time) })
	return points, nil
}

// change decodes a percent like "-21.0%".
type change float64

func (c *change) UnmarshalJSON(b []byte) error {
	var v struct {
		Change string `json:"change"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(v.Change, "%"), 64)
	if err != nil {
		return fmt.Errorf("ge: parse change %q: %w", v.Change, err)
	}
	*c = change(f)
	return nil
}

// parseShortPrice decodes a price that Jagex sends either as a number or
// as display text like "800.4k", "1.4b", "- 13.5k", or "+5".
func parseShortPrice(raw json.RawMessage) (int, error) {
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("ge: parse price %s: %w", raw, err)
	}
	text := strings.NewReplacer(" ", "", ",", "", "+", "").Replace(s)
	mult := 1.0
	switch {
	case strings.HasSuffix(text, "k"):
		mult = 1e3
	case strings.HasSuffix(text, "m"):
		mult = 1e6
	case strings.HasSuffix(text, "b"):
		mult = 1e9
	}
	f, err := strconv.ParseFloat(strings.TrimRight(text, "kmb"), 64)
	if err != nil {
		return 0, fmt.Errorf("ge: parse price %q: %w", s, err)
	}
	return int(f * mult), nil
}
