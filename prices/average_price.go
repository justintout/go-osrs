package prices

import (
	"context"
	"fmt"
	"time"

	"github.com/justintout/go-osrs/internal/transport"
)

// FiveMinute returns the latest 5-minute average prices and volumes for
// all items.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#5-minute_prices
func (c *Client) FiveMinute(ctx context.Context) (Averages, error) {
	return c.averages(ctx, "/5m")
}

// FiveMinuteStartingAt returns 5-minute average prices and volumes for the
// window starting at t, which must fall on a 5-minute boundary.
func (c *Client) FiveMinuteStartingAt(ctx context.Context, t time.Time) (Averages, error) {
	return c.AveragesAt(ctx, FiveMinute, t)
}

// AveragesAt returns average prices and volumes for every item in the
// window of the given timestep starting at t, which must fall on that
// timestep's boundary. Data goes back to March 2021. Windows that have
// closed never change, so with a cache they are fetched only once.
func (c *Client) AveragesAt(ctx context.Context, step Timestep, t time.Time) (Averages, error) {
	d, err := step.Duration()
	if err != nil {
		return nil, err
	}
	if t.Unix()%int64(d.Seconds()) != 0 {
		return nil, fmt.Errorf("prices: %s is not on a %s boundary", t, step)
	}
	return c.averages(ctx, fmt.Sprintf("/%s?timestamp=%d", step, t.Unix()), closed(t, d)...)
}

// OneHour returns the latest hourly average prices and volumes for all
// items.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#1-hour_prices
func (c *Client) OneHour(ctx context.Context) (Averages, error) {
	return c.averages(ctx, "/1h")
}

// OneHourStartingAt returns hourly average prices and volumes for the hour
// starting at t, which must fall on an hour boundary.
func (c *Client) OneHourStartingAt(ctx context.Context, t time.Time) (Averages, error) {
	return c.AveragesAt(ctx, OneHour, t)
}

// closed keeps averages for a window that ended over ten minutes ago
// cached for good, since trades in a past window never change.
func closed(start time.Time, window time.Duration) []transport.Option {
	if time.Since(start.Add(window)) < 10*time.Minute {
		return nil
	}
	return []transport.Option{transport.MinTTL(100 * 365 * 24 * time.Hour)}
}

func (c *Client) averages(ctx context.Context, path string, opts ...transport.Option) (Averages, error) {
	var a averageResponse
	if err := c.t.GetJSON(ctx, c.baseURL+path, &a, opts...); err != nil {
		return nil, err
	}
	return a.Data, nil
}

// Averages is a map of average prices, keyed with item IDs
type Averages map[int]Average

// Average is a struct of average high/low prices and volumes for some time period
type Average struct {
	AvgHighPrice    int `json:"avgHighPrice"`
	HighPriceVolume int `json:"highPriceVolume"`
	AvgLowPrice     int `json:"avgLowPrice"`
	LowPriceVolume  int `json:"lowPriceVolume"`
}

type averageResponse struct {
	Data Averages `json:"data"`
}
