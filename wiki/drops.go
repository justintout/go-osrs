package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Drop is one line of a monster's drop table.
type Drop struct {
	// Page is the wiki page whose drop table lists this line.
	Page string `json:"page"`
	Item string `json:"item"`
	// Source is the dropping page and version, such as "Vorkath#Post-quest".
	Source string `json:"source"`
	// Rarity is the wiki's text, such as "1/5,000", "Always", or "Rare".
	Rarity string `json:"rarity"`
	// Probability is the per-roll drop chance parsed from Rarity. It is 0
	// when Rarity is vague, such as "Rare" or "Varies".
	Probability  float64 `json:"probability"`
	QuantityLow  int     `json:"quantity_low"`
	QuantityHigh int     `json:"quantity_high"`
	// Rolls is how many times the table is rolled per kill. A few tables
	// give an average, such as 1.7.
	Rolls         float64 `json:"rolls"`
	RareDropTable bool    `json:"rare_drop_table"`
}

// Drops returns the drop table on a monster's wiki page, across all of its
// versions.
func (c *Client) Drops(ctx context.Context, monsterPage string) ([]Drop, error) {
	return c.drops(ctx, "page_name", monsterPage)
}

// DropSources returns every drop table line that drops the named item.
func (c *Client) DropSources(ctx context.Context, item string) ([]Drop, error) {
	return c.drops(ctx, "item_name", item)
}

// AllDrops returns every drop table line on the wiki, about 40,000 rows
// in 8 requests. Use it for scans across many monsters instead of calling
// Drops for each one.
func (c *Client) AllDrops(ctx context.Context) ([]Drop, error) {
	rows, err := runPaged[dropRow](ctx, c, Bucket("dropsline").Select("page_name", "drop_json", "rare_drop_table"))
	if err != nil {
		return nil, err
	}
	return parseDrops(rows)
}

type dropRow struct {
	Page string `json:"page_name"`
	JSON string `json:"drop_json"`
	RDT  Flag   `json:"rare_drop_table"`
}

func (c *Client) drops(ctx context.Context, field, value string) ([]Drop, error) {
	var rows []dropRow
	q := Bucket("dropsline").Select("page_name", "drop_json", "rare_drop_table").Where(field, value).Limit(maxRows)
	if err := c.Run(ctx, q, &rows); err != nil {
		return nil, err
	}
	return parseDrops(rows)
}

func parseDrops(rows []dropRow) ([]Drop, error) {
	drops := make([]Drop, 0, len(rows))
	for _, r := range rows {
		var d struct {
			Item   string  `json:"Dropped item"`
			Source string  `json:"Dropped from"`
			Rarity string  `json:"Rarity"`
			Low    flexNum `json:"Quantity Low"`
			High   flexNum `json:"Quantity High"`
			Rolls  flexNum `json:"Rolls"`
		}
		if err := json.Unmarshal([]byte(r.JSON), &d); err != nil {
			return nil, fmt.Errorf("wiki: decode drop_json: %w", err)
		}
		p, _ := ParseRarity(d.Rarity)
		drops = append(drops, Drop{
			Page: r.Page, Item: d.Item, Source: d.Source, Rarity: d.Rarity, Probability: p,
			QuantityLow: int(d.Low), QuantityHigh: int(d.High), Rolls: float64(d.Rolls), RareDropTable: bool(r.RDT),
		})
	}
	return drops, nil
}

// ParseRarity converts wiki rarity text to a probability. It handles
// fractions like "1/5,000" or "3/128" and "Always". It reports false for
// vague text like "Rare" or "Varies".
func ParseRarity(rarity string) (float64, bool) {
	s := strings.ReplaceAll(strings.TrimSpace(rarity), ",", "")
	if strings.EqualFold(s, "Always") {
		return 1, true
	}
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		return 0, false
	}
	n, err1 := strconv.ParseFloat(strings.TrimPrefix(num, "~"), 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d <= 0 {
		return 0, false
	}
	return n / d, true
}
