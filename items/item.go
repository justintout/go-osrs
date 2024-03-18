package items

import "encoding/json"

// Mapping returns a slice of all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Mapping
func (c *Client) Mapping() ([]Item, error) {
	res, err := c.httpClient.Get("https" + c.baseURL + "/mapping")
	if err != nil {
		return nil, err
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
	Lowalch  int    `json:"lowalch"`
	Limit    int    `json:"limit"`
	Value    int    `json:"value"`
	Highalch int    `json:"highalch"`
	Icon     string `json:"icon"`
	Name     string `json:"name"`
}
