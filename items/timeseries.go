package items

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Timeseries returns a list of average high/low price and volume for a given id at the given timestep
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Time-series
func (c *Client) Timeseries(id int, timestep Timestep) (Timeseries, error) {
	res, err := c.httpClient.Get(fmt.Sprintf("https://%s/timeseries?timestep=%s&id=%d", c.baseURL, timestep, id))
	if err != nil {
		return Timeseries{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Timeseries{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var t Timeseries
	err = json.NewDecoder(res.Body).Decode(&t)
	return t, err
}

// Timestep is a supported timeseries timestep value
type Timestep string

const (
	// FiveMinute is the 5m timestep, returning a timeseries in 5 minute increments
	FiveMinute Timestep = "5m"
	// OneHour is the 1h timestep, returning a timeseries in 1 hour increments
	OneHour Timestep = "1h"
	// SixHour is the 6h timestep, returning a timeseries in 6 hour increments
	SixHour Timestep = "6h"
	// TwentyFourHour is the 24h timestep, returning a timeseries in 24 hour increments
	TwentyFourHour Timestep = "24h"
)

// Timeseries is a struct representing a timeseries of average low/high prices and volumes for a given ItemID
type Timeseries struct {
	ItemID int                   `json:"itemId"`
	Data   []TimeseriesDatapoint `json:"data"`
}

// TimeseriesDatapoint is a particular datapoint within a timeseries
type TimeseriesDatapoint struct {
	Timestamp       time.Time
	AvgHighPrice    int
	HighPriceVolume int
	AvgLowPrice     int
	LowPriceVolume  int
}

// UnmarshalJSON implements the JSONUnmarshaler interface
// We use custom unmashaling to get timestamps as Go time.Time objects and not Unix timestamps
func (td *TimeseriesDatapoint) UnmarshalJSON(data []byte) error {
	var t struct {
		T   int64 `json:"timestamp"`
		AHP int   `json:"avgHighPrice"`
		HPV int   `json:"highPriceVolume"`
		ALP int   `json:"avgLowPrice"`
		LPV int   `json:"lowPriceVolume"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	td.Timestamp = time.Unix(t.T, 0)
	td.AvgHighPrice = t.AHP
	td.HighPriceVolume = t.HPV
	td.AvgLowPrice = t.ALP
	td.LowPriceVolume = t.LPV
	return nil
}
