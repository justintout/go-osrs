package items

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Mapping returns a slice of all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Mapping
func (c *Client) Mapping() ([]Item, error) {
	res, err := c.httpClient.Get("https://" + c.baseURL + "/mapping")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var items []Item
	err = json.NewDecoder(res.Body).Decode(&items)
	return items, err
}

// Item is the representation of an item
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
