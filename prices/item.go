package prices

import "context"

// Mapping returns metadata for every tradeable item.
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Mapping
func (c *Client) Mapping(ctx context.Context) ([]Item, error) {
	var items []Item
	err := c.t.GetJSON(ctx, c.baseURL+"/mapping", &items)
	return items, err
}

// Item is metadata for a tradeable item.
type Item struct {
	Examine  string `json:"examine"`
	ID       int    `json:"id"`
	Members  bool   `json:"members"`
	LowAlch  int    `json:"lowalch"`
	Limit    int    `json:"limit"`
	Value    int    `json:"value"`
	HighAlch int    `json:"highalch"`
	Icon     string `json:"icon"`
	Name     string `json:"name"`
}
