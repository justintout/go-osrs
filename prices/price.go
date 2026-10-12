package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Latest returns the latest price spreads for all items.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Latest_price_(all_items)
func (c *Client) Latest(ctx context.Context) (Prices, error) {
	var p priceResponse
	err := c.t.GetJSON(ctx, c.baseURL+"/latest", &p)
	return p.Data, err
}

// LatestFor returns the latest price spread for one item.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Query_parameters
func (c *Client) LatestFor(ctx context.Context, id int) (Spread, error) {
	var p priceResponse
	err := c.t.GetJSON(ctx, fmt.Sprintf("%s/latest?id=%d", c.baseURL, id), &p)
	if err != nil {
		return Spread{}, err
	}
	return p.Data[id], nil
}

// Prices is a map of item price spreads, keyed to item IDs
type Prices map[int]Spread

// Spread is the latest instant-buy (High) and instant-sell (Low) price
// for an item. A price is 0 when the item has no trade on that side.
type Spread struct {
	High     int
	HighTime time.Time
	Low      int
	LowTime  time.Time
}

// UnmarshalJSON implements the JSONUnmarshaler interface
// We use custom unmarshaling to make our timestamps Go time.Time structs, instead of Unix timestamps
func (s *Spread) UnmarshalJSON(data []byte) error {
	var rs struct {
		H  int   `json:"high"`
		HT int64 `json:"highTime"`
		L  int   `json:"low"`
		LT int64 `json:"lowTime"`
	}
	if err := json.Unmarshal(data, &rs); err != nil {
		return err
	}
	s.High = rs.H
	s.HighTime = time.Unix(rs.HT, 0)
	s.Low = rs.L
	s.LowTime = time.Unix(rs.LT, 0)
	return nil
}

type priceResponse struct {
	Data Prices `json:"data"`
}
