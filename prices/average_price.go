package prices

import (
	"context"
	"fmt"
	"time"
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
	if t.Unix()%300 != 0 {
		return nil, fmt.Errorf("prices: %s is not on a 5-minute boundary", t)
	}
	return c.averages(ctx, fmt.Sprintf("/5m?timestamp=%d", t.Unix()))
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
	if t.Unix()%3600 != 0 {
		return nil, fmt.Errorf("prices: %s is not on an hour boundary", t)
	}
	return c.averages(ctx, fmt.Sprintf("/1h?timestamp=%d", t.Unix()))
}

func (c *Client) averages(ctx context.Context, path string) (Averages, error) {
	var a averageResponse
	if err := c.t.GetJSON(ctx, c.baseURL+path, &a); err != nil {
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
