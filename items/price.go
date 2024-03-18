package items

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Latest returns the latest price spreads for all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Latest_price_(all_items)
func (c *Client) Latest() (Prices, error) {
	res, err := c.httpClient.Get("https://" + c.baseURL + "/latest")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var p priceResponse
	err = json.NewDecoder(res.Body).Decode(&p)
	if err != nil {
		return nil, err
	}
	return p.Data, nil
}

// LatestFor returns the latest price spread for the given item ID
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Query_parameters
func (c *Client) LatestFor(id int) (Spread, error) {
	res, err := c.httpClient.Get(fmt.Sprintf("https://%s/latest?id=%d", c.baseURL, id))
	if err != nil {
		return Spread{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Spread{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var p priceResponse
	err = json.NewDecoder(res.Body).Decode(&p)
	if err != nil {
		return Spread{}, err
	}
	return p.Data[id], nil
}

// Prices is a map of item price spreads, keyed to item IDs
type Prices map[int]Spread

// UnmarshalJSON implements the JSONUnmarshaler interface
// We use custom unmarshaling to make our item IDs integers instead of the strings returned by the API
func (p *Prices) UnmarshalJSON(data []byte) error {
	var items map[string]Spread
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	*p = make(Prices)
	for k, v := range items {
		id, err := strconv.Atoi(k)
		if err != nil {
			return err
		}
		(*p)[id] = v
	}
	return nil
}

// Spread is a price spread for an item
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
