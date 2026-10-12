package profit

import (
	"sort"
	"strings"

	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

// ShopFlip is the profit from buying an item in an NPC shop and selling
// it on the Grand Exchange.
type ShopFlip struct {
	Store    string `json:"store"`
	Location string `json:"location,omitempty"`
	Item     string `json:"item"`
	ItemID   int    `json:"item_id"`
	Members  bool   `json:"members"`
	// ShopPrice is the price of the first unit at full stock.
	ShopPrice int `json:"shop_price"`
	// SellPrice is the Grand Exchange instant-sell price after tax.
	SellPrice int `json:"sell_price"`
	// Units is how many to buy in one visit: until the rising shop price
	// meets SellPrice, stock runs out, or a day of GE volume is reached.
	// Shops with unlimited stock count one inventory of 28.
	Units       int    `json:"units"`
	Cost        int    `json:"cost"`
	Profit      int    `json:"profit"`
	Stock       int    `json:"stock"`
	RestockTime string `json:"restock_time,omitempty"`
	Volume24h   int    `json:"volume_24h"`
}

const inventorySize = 28

// ShopFlips returns coin shops whose items sell on the Grand Exchange for
// more than they cost, most profitable visit first.
//
// Each unit bought raises the shop price by Delta tenths of a percent of
// the item's value, as the wiki describes shop pricing. Items that traded
// fewer than minVolume times in the last 24 hours are skipped, since
// their prices are often stale.
func ShopFlips(stores []wiki.StoreItem, catalog *prices.Catalog, latest prices.Prices, volumes map[int]int, minVolume int) []ShopFlip {
	var out []ShopFlip
	seen := map[string]bool{}
	for _, s := range stores {
		// Some shops list an item twice.
		key := s.Store + "|" + s.Item
		if seen[key] {
			continue
		}
		seen[key] = true
		// Reward shops price in points even when the wiki says coins.
		reward := strings.Contains(s.Store, "Reward")
		item, ok := catalog.ByName(s.Item)
		if !ok || s.Currency != "Coins" || s.Stock == 0 || reward || volumes[item.ID] < minVolume {
			continue
		}
		sell := prices.AfterTax(item.ID, latest[item.ID].Low)
		if sell <= s.Price {
			continue
		}
		limit := min(volumes[item.ID], inventorySize)
		if s.Stock > 0 {
			limit = min(volumes[item.ID], s.Stock)
		}
		f := ShopFlip{
			Store: s.Store, Location: s.Location, Item: s.Item, ItemID: item.ID, Members: s.Members,
			ShopPrice: s.Price, SellPrice: sell, Stock: s.Stock, RestockTime: s.RestockTime, Volume24h: volumes[item.ID],
		}
		for k := 0; k < limit; k++ {
			price := unitPrice(s, item.Value, k)
			if price >= sell {
				break
			}
			f.Units++
			f.Cost += price
		}
		f.Profit = f.Units*sell - f.Cost
		if f.Profit > 0 {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Profit > out[b].Profit })
	return out
}

// unitPrice is the shop price after bought units have left the stock.
// Shops without a multiplier on the wiki keep their listed price.
func unitPrice(s wiki.StoreItem, value, bought int) int {
	if s.SellMultiplier == 0 || value == 0 {
		return s.Price
	}
	return max(value*(s.SellMultiplier+s.Delta*bought)/1000, s.Price)
}
