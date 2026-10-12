package wiki

import (
	"context"
	"regexp"
	"strings"
)

// LogItem is a collection log item and where it comes from.
type LogItem struct {
	Item    string      `json:"item"`
	ItemID  int         `json:"item_id"`
	Sources []LogSource `json:"sources"`
}

// LogSource is one way to get a collection log item.
type LogSource struct {
	// Page is the source's wiki page, such as "Vorkath".
	Page string `json:"page"`
	// Version is the page section, such as "Post-quest", or "".
	Version string `json:"version,omitempty"`
	Rarity  string `json:"rarity"`
	// Probability is parsed from Rarity, or 0 when Rarity is vague.
	Probability float64 `json:"probability"`
}

var pageLinkRE = regexp.MustCompile(`\[\[([^\]|]+)`)

// CollectionLog returns every collection log item with its sources and
// rates.
func (c *Client) CollectionLog(ctx context.Context) ([]LogItem, error) {
	rows, err := runPaged[struct {
		Item    string   `json:"item_name"`
		ItemID  int      `json:"item_id"`
		Sources []string `json:"sources"`
		Rates   []string `json:"rates"`
	}](ctx, c, Bucket("collection_log_source").Select("item_name", "item_id", "sources", "rates"))
	if err != nil {
		return nil, err
	}
	items := make([]LogItem, 0, len(rows))
	for _, r := range rows {
		item := LogItem{Item: r.Item, ItemID: r.ItemID}
		for i, src := range r.Sources {
			m := pageLinkRE.FindStringSubmatch(src)
			if m == nil {
				continue
			}
			page, version, _ := strings.Cut(m[1], "#")
			s := LogSource{Page: strings.TrimSpace(page), Version: version}
			if i < len(r.Rates) {
				s.Rarity = r.Rates[i]
				s.Probability, _ = ParseRarity(r.Rates[i])
			}
			item.Sources = append(item.Sources, s)
		}
		items = append(items, item)
	}
	return items, nil
}
