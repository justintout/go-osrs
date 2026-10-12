package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/justintout/go-osrs/ge"
	"github.com/justintout/go-osrs/prices"
)

type itemIn struct {
	Item string `json:"item" arg:"" jsonschema:"item name, such as Abyssal whip"`
}

type priceOut struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Buy is the latest instant-buy price and Sell the latest instant-sell
	// price. Margin is Buy after tax minus Sell: the profit from buying at
	// Sell and selling at Buy.
	Buy        int       `json:"buy"`
	BuyTime    time.Time `json:"buy_time"`
	Sell       int       `json:"sell"`
	SellTime   time.Time `json:"sell_time"`
	Tax        int       `json:"tax"`
	Margin     int       `json:"margin"`
	Volume24h  int       `json:"volume_24h"`
	BuyLimit   int       `json:"buy_limit"`
	HighAlch   int       `json:"high_alch"`
	Members    bool      `json:"members"`
	PriceNotes string    `json:"price_notes,omitempty"`
}

func (o priceOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (id %d)\n", o.Name, o.ID)
	t := newTable(&b)
	t.row("instant buy", num(int64(o.Buy))+" gp", ago(o.BuyTime))
	t.row("instant sell", num(int64(o.Sell))+" gp", ago(o.SellTime))
	t.row("tax", num(int64(o.Tax))+" gp")
	t.row("margin after tax", num(int64(o.Margin))+" gp")
	t.row("24h volume", num(int64(o.Volume24h)))
	t.row("buy limit", num(int64(o.BuyLimit)))
	t.row("high alch", num(int64(o.HighAlch))+" gp")
	t.flush()
	if o.PriceNotes != "" {
		b.WriteString(o.PriceNotes + "\n")
	}
	return b.String()
}

var priceCmd = newCommand("price", "get an item's live Grand Exchange buy and sell prices, tax, margin, volume, and buy limit",
	func(ctx context.Context, d *deps, in itemIn) (priceOut, error) {
		if err := need("item", in.Item); err != nil {
			return priceOut{}, err
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return priceOut{}, err
		}
		spread, err := d.prices.LatestFor(ctx, item.ID)
		if err != nil {
			return priceOut{}, err
		}
		vols, _, err := d.prices.Volumes(ctx)
		if err != nil {
			return priceOut{}, err
		}
		out := priceOut{
			ID: item.ID, Name: item.Name, Buy: spread.High, BuyTime: spread.HighTime, Sell: spread.Low, SellTime: spread.LowTime,
			Tax: prices.Tax(item.ID, spread.High), Margin: prices.AfterTax(item.ID, spread.High) - spread.Low,
			Volume24h: vols[item.ID], BuyLimit: item.Limit, HighAlch: item.HighAlch, Members: item.Members,
		}
		if spread.High == 0 || spread.Low == 0 {
			out.PriceNotes = "no recent trade on one side; margin is not meaningful"
		}
		return out, nil
	})

type guidePriceIn struct {
	Item    string `json:"item" arg:"" jsonschema:"item name"`
	History bool   `json:"history,omitempty" jsonschema:"include 180 days of daily guide prices"`
}

type guidePriceOut struct {
	ge.Item
	History []ge.Point `json:"history,omitempty"`
}

func (o guidePriceOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s guide price %s gp (today %s)\n30d %+.1f%%  90d %+.1f%%  180d %+.1f%%\n",
		o.Name, num(int64(o.Price)), signed(o.TodayChange), o.Change30d, o.Change90d, o.Change180d)
	if len(o.History) > 0 {
		b.WriteString("\n")
		t := newTable(&b, "DATE", "PRICE", "30D AVG")
		for _, p := range o.History {
			t.row(p.Time.UTC().Format(time.DateOnly), num(int64(p.Price)), num(int64(p.Average)))
		}
		t.flush()
	}
	return b.String()
}

var guidePriceCmd = newCommand("guide-price", "get an item's official Jagex guide price and 30/90/180-day trend",
	func(ctx context.Context, d *deps, in guidePriceIn) (guidePriceOut, error) {
		if err := need("item", in.Item); err != nil {
			return guidePriceOut{}, err
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return guidePriceOut{}, err
		}
		g, err := d.ge.Item(ctx, item.ID)
		if err != nil {
			return guidePriceOut{}, err
		}
		out := guidePriceOut{Item: *g}
		if in.History {
			out.History, err = d.ge.History(ctx, item.ID)
		}
		return out, err
	})

func ago(t time.Time) string {
	if t.Unix() <= 0 {
		return "never"
	}
	return time.Since(t).Round(time.Minute).String() + " ago"
}
