package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/justintout/go-osrs/backtest"
	"github.com/justintout/go-osrs/market"
	"github.com/justintout/go-osrs/prices"
)

type backtestIn struct {
	Item          string  `json:"item,omitempty" arg:"" jsonschema:"item to backtest; omit to backtest the top daily-cycle candidates"`
	Days          int     `json:"days,omitempty" jsonschema:"days of hourly history, 42 by default: hours are learned on the first two thirds and traded on the last third"`
	BuyHour       int     `json:"buy_hour,omitempty" jsonschema:"UTC hour to buy, instead of the learned one"`
	SellHour      int     `json:"sell_hour,omitempty" jsonschema:"UTC hour to sell, instead of the learned one"`
	Capital       int     `json:"capital,omitempty" jsonschema:"most coins to tie up, no cap by default"`
	Participation float64 `json:"participation,omitempty" jsonschema:"share of each hour's volume your orders can take, 0.1 by default"`
	Top           int     `json:"top,omitempty" jsonschema:"candidates to backtest when no item is given, 10 by default"`
}

type backtestLine struct {
	Item     string             `json:"item"`
	ItemID   int                `json:"item_id"`
	BuyHour  int                `json:"buy_hour"`
	SellHour int                `json:"sell_hour"`
	From     time.Time          `json:"trading_from"`
	Result   backtest.Result    `json:"result"`
	Cycle    *market.DailyCycle `json:"cycle,omitempty"`
}

type backtestOut struct {
	Lines   []backtestLine `json:"backtests"`
	Fetched int            `json:"fetched_windows,omitempty"`
}

func (o backtestOut) text() string {
	var b strings.Builder
	if o.Fetched > 0 {
		fmt.Fprintf(&b, "archived %d new hourly windows\n\n", o.Fetched)
	}
	t := newTable(&b, "ITEM", "BUY (UTC)", "SELL (UTC)", "TRADES", "PROFIT", "MAX CAPITAL", "ROI", "WINNING DAYS", "HELD AT END")
	for _, l := range o.Lines {
		r := l.Result
		t.row(l.Item, fmt.Sprintf("%02d:00", l.BuyHour), fmt.Sprintf("%02d:00", l.SellHour), fmt.Sprint(len(r.Trades)), signed(r.Profit),
			num(int64(r.MaxCapital)), fmt.Sprintf("%+.1f%%", r.ROI*100), fmt.Sprintf("%d/%d", r.WinningDays, r.Days), signed(r.Unrealized))
	}
	t.flush()
	if len(o.Lines) > 0 {
		fmt.Fprintf(&b, "\ntraded from %s UTC; hours were learned on earlier data.\n", o.Lines[0].From.Format("Jan 02"))
	}
	b.WriteString("fills: buys only when the hour's average instant-sell price reached the limit, sells likewise, each up to the participation share of volume; tax and buy limits apply.\n")
	return b.String()
}

var backtestCmd = newCommand("backtest", "backtest buying at an item's cheap hour and selling at its dear hour, with conservative fills, tax, and buy limits, on days the hours were not learned from",
	func(ctx context.Context, d *deps, in backtestIn) (backtestOut, error) {
		days := orDefault(in.Days, 42)
		if days < 9 || days > 120 {
			return backtestOut{}, fmt.Errorf("days must be between 9 and 120")
		}
		a, err := d.priceArchive()
		if err != nil {
			return backtestOut{}, err
		}
		to := time.Now().UTC().Truncate(24 * time.Hour)
		from := to.Add(-time.Duration(days) * 24 * time.Hour)
		fetched, err := a.Ensure(ctx, prices.OneHour, from, to)
		if err != nil {
			return backtestOut{}, err
		}
		series, err := a.Series(prices.OneHour, from, to)
		if err != nil {
			return backtestOut{}, err
		}
		cat, err := d.itemCatalog(ctx)
		if err != nil {
			return backtestOut{}, err
		}
		split := from.Add(time.Duration(days*2/3) * 24 * time.Hour)
		train := map[int][]prices.TimeseriesDatapoint{}
		for id, s := range series {
			for i, p := range s {
				if !p.Timestamp.Before(split) {
					train[id] = s[:i]
					break
				}
			}
		}

		var ids []int
		if in.Item != "" {
			item, err := d.findItem(ctx, in.Item)
			if err != nil {
				return backtestOut{}, err
			}
			ids = []int{item.ID}
		} else {
			latest, err := d.prices.Latest(ctx)
			if err != nil {
				return backtestOut{}, err
			}
			for _, c := range market.DailyCycles(train, cat, 6) {
				if c.Edge > 0 && c.Consistency >= 0.6 && c.Volume >= 1000 && latest[c.ItemID].Low >= 100 {
					ids = append(ids, c.ItemID)
				}
			}
			ids = top(ids, orDefault(in.Top, 10))
		}

		out := backtestOut{Fetched: fetched}
		for _, id := range ids {
			item, _ := cat.ByID(id)
			line := backtestLine{Item: item.Name, ItemID: id, BuyHour: in.BuyHour, SellHour: in.SellHour, From: split}
			if in.BuyHour == 0 && in.SellHour == 0 {
				cycles := market.DailyCycles(map[int][]prices.TimeseriesDatapoint{id: train[id]}, cat, 6)
				if len(cycles) == 0 {
					return backtestOut{}, fmt.Errorf("%s has no daily pattern in the training days; pass buy_hour and sell_hour to test one anyway", item.Name)
				}
				line.BuyHour, line.SellHour, line.Cycle = cycles[0].LowHour, cycles[0].HighHour, &cycles[0]
			}
			strategy := backtest.Cycle{BuyHour: line.BuyHour, SellHour: line.SellHour, Quantity: max(item.Limit, 1)}
			cfg := backtest.Config{ItemID: id, BuyLimit: item.Limit, Participation: in.Participation, Capital: in.Capital, From: split}
			line.Result = backtest.Run(series[id], strategy, cfg)
			out.Lines = append(out.Lines, line)
		}
		return out, nil
	})
