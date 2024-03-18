package items

// Mapping returns a slice of all items
// https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Mapping
func (c *Client) Mapping() ([]Item, error) {
	var items []Item
	err := c.get("https://"+c.baseURL+"/mapping", &items)
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
