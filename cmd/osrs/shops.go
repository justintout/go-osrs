package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/justintout/go-osrs/profit"
)

type shopsIn struct {
	Item      string `json:"item,omitempty" jsonschema:"only shops selling this item"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum 24h GE trades for an item to count, 500 by default"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
	F2P       bool   `json:"f2p,omitempty" jsonschema:"free-to-play shops only"`
}

type shopsOut struct {
	Flips []profit.ShopFlip `json:"flips"`
}

func (o shopsOut) text() string {
	var b strings.Builder
	t := newTable(&b, "ITEM", "SHOP", "LOCATION", "SHOP PRICE", "GE SELL", "UNITS", "PROFIT/VISIT", "24H VOL")
	for _, f := range o.Flips {
		t.row(f.Item, f.Store, f.Location, num(int64(f.ShopPrice)), num(int64(f.SellPrice)), num(int64(f.Units)), num(int64(f.Profit)), num(int64(f.Volume24h)))
	}
	t.flush()
	b.WriteString("\nshop prices rise with each unit bought; GE sell is the instant-sell price after tax\n")
	return b.String()
}

var shopsCmd = newCommand("shops", "find NPC shop items that sell on the Grand Exchange for more than they cost, with units worth buying per visit",
	func(ctx context.Context, d *deps, in shopsIn) (shopsOut, error) {
		stores, err := d.wiki.Stores(ctx)
		if err != nil {
			return shopsOut{}, err
		}
		cat, latest, vols, err := market(ctx, d)
		if err != nil {
			return shopsOut{}, err
		}
		var out []profit.ShopFlip
		for _, f := range profit.ShopFlips(stores, cat, latest, vols, orDefault(in.MinVolume, defaultMinVolume)) {
			if (in.F2P && f.Members) || (in.Item != "" && !strings.EqualFold(f.Item, in.Item)) {
				continue
			}
			out = append(out, f)
		}
		if len(out) == 0 && in.Item != "" {
			return shopsOut{}, fmt.Errorf("no shop sells %q below its GE price", in.Item)
		}
		return shopsOut{Flips: top(out, orDefault(in.Top, 20))}, nil
	})
