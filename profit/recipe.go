package profit

import (
	"sort"

	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

// RecipeProfit is the margin on making one batch of a recipe.
type RecipeProfit struct {
	Recipe wiki.Recipe `json:"recipe"`
	// Cost buys every material at its instant-buy price.
	Cost int `json:"cost"`
	// Revenue sells the output at its instant-sell price, after tax.
	Revenue int `json:"revenue"`
	Profit  int `json:"profit"`
	// ProfitPerHour assumes one action every Recipe.Ticks ticks with no
	// banking. It is 0 when the wiki gives no tick count.
	ProfitPerHour   int `json:"profit_per_hour"`
	OutputVolume24h int `json:"output_volume_24h"`
}

// Recipes prices every recipe whose output and materials all trade on the
// Grand Exchange and returns those with a positive profit, best first.
// It buys and sells instantly, so margins hold without waiting on offers.
//
// Thinly traded items have stale last-trade prices that produce fake
// margins, so the output and every material must have traded at least
// minVolume times in the last 24 hours.
func Recipes(recipes []wiki.Recipe, catalog *prices.Catalog, latest prices.Prices, volumes map[int]int, minVolume int) []RecipeProfit {
	var out []RecipeProfit
	for _, r := range recipes {
		outItem, ok := catalog.ByName(r.Output.Name)
		if !ok || r.Output.Quantity <= 0 {
			continue
		}
		sell := latest[outItem.ID].Low
		if sell == 0 || volumes[outItem.ID] < minVolume {
			continue
		}
		cost, priced := 0.0, true
		for _, m := range r.Materials {
			item, ok := catalog.ByName(m.Name)
			buy := latest[item.ID].High
			if !ok || buy == 0 || volumes[item.ID] < minVolume {
				priced = false
				break
			}
			cost += float64(buy) * m.Quantity
		}
		if !priced || len(r.Materials) == 0 {
			continue
		}
		revenue := float64(prices.AfterTax(outItem.ID, sell)) * r.Output.Quantity
		p := RecipeProfit{Recipe: r, Cost: int(cost), Revenue: int(revenue), Profit: int(revenue - cost), OutputVolume24h: volumes[outItem.ID]}
		if p.Profit <= 0 {
			continue
		}
		if r.Ticks > 0 {
			p.ProfitPerHour = int(float64(p.Profit) * 3600 / (r.Ticks * 0.6))
		}
		out = append(out, p)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Profit > out[b].Profit })
	return out
}
