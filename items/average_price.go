package items

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FiveMinute returns 5-minute average high/low prices and volume for all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#5-minute_prices
func (c *Client) FiveMinute() (Averages, error) {
	var a averageResponse
	err := c.get(c.baseURL+"/5m", &a)
	if err != nil {
		return nil, err
	}
	return a.Data, nil
}

// FiveMinuteStartingAt returns 5-minute average high/low prices and volumes for all items, starting at the given time
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Query_parameters_2
func (c *Client) FiveMinuteStartingAt(t time.Time) (Averages, error) {
	// TODO: timestamp must be divisible by 3600
	//       either find closest past stamp divisible by 3600 or error
	var a averageResponse
	err := c.get(fmt.Sprintf("%s/5m?timestamp=%d", c.baseURL, t.Unix()), &a)
	if err != nil {
		return nil, err
	}
	return a.Data, nil
}

// OneHour returns hourly average high/low prices and volumes for all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#1-hour_prices
func (c *Client) OneHour() (Averages, error) {
	var a averageResponse
	err := c.get(c.baseURL+"/1h", &a)
	if err != nil {
		return nil, err
	}
	return a.Data, nil
}

// OneHour returns hourly average high/low prices and volumes for all items, starting at the given time
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Query_parameters_3
func (c *Client) OneHourStartingAt(t time.Time) (Averages, error) {
	// TODO: timestamp must be divisible by 3600
	//       either find closest past stamp divisible by 3600 or error
	var a averageResponse
	err := c.get(fmt.Sprintf("%s/1h?timestamp=%d", c.baseURL, t.Unix()), &a)
	if err != nil {
		return nil, err
	}
	return a.Data, nil
}

// Averages is a map of average prices, keyed with item IDs
type Averages map[int]Average

// UnmarshalJSON implements the JSONUnmarshaler interface
// We use custom unmarshaling to make our item IDs integers instead of the strings returned by the API
func (a *Averages) UnmarshalJSON(data []byte) error {
	var items map[string]Average
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	*a = make(Averages)
	for k, v := range items {
		id, err := strconv.Atoi(k)
		if err != nil {
			return err
		}
		(*a)[id] = v
	}
	return nil
}

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
