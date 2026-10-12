// Package profit finds money-making margins from live Grand Exchange
// prices: high alchemy and item recipes.
package profit

import (
	"sort"

	"github.com/justintout/go-osrs/prices"
)

// NatureRuneID is the item ID of the nature rune each high alch consumes.
const NatureRuneID = 561

// Alch is the profit from buying an item and casting High Level Alchemy
// on it.
type Alch struct {
	ItemID   int    `json:"item_id"`
	Name     string `json:"name"`
	BuyPrice int    `json:"buy_price"`
	HighAlch int    `json:"high_alch"`
	// Profit is per cast, after the nature rune.
	Profit   int `json:"profit"`
	BuyLimit int `json:"buy_limit"`
	// ProfitPerLimit is Profit times the 4-hour buy limit.
	ProfitPerLimit int  `json:"profit_per_limit"`
	Volume24h      int  `json:"volume_24h"`
	Members        bool `json:"members"`
}

// Alchs returns every item with a positive high alch profit, best first by
// profit per buy limit. It buys items and nature runes at the instant-buy
// price, so margins are what a player gets without waiting on offers.
// Items that traded fewer than minVolume times in the last 24 hours are
// skipped, since their last-trade prices are often stale.
func Alchs(items []prices.Item, latest prices.Prices, volumes map[int]int, minVolume int) []Alch {
	nature := latest[NatureRuneID].High
	var out []Alch
	for _, i := range items {
		buy := latest[i.ID].High
		if buy == 0 || i.HighAlch == 0 || volumes[i.ID] < minVolume {
			continue
		}
		p := i.HighAlch - buy - nature
		if p <= 0 {
			continue
		}
		out = append(out, Alch{
			ItemID: i.ID, Name: i.Name, BuyPrice: buy, HighAlch: i.HighAlch, Profit: p,
			BuyLimit: i.Limit, ProfitPerLimit: p * i.Limit, Volume24h: volumes[i.ID], Members: i.Members,
		})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ProfitPerLimit > out[b].ProfitPerLimit })
	return out
}
