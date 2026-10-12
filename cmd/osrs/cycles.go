package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/justintout/go-osrs/market"
	"github.com/justintout/go-osrs/prices"
)

type cyclesIn struct {
	Item      string `json:"item,omitempty" arg:"" jsonschema:"one item to profile; omit to scan the whole market"`
	Days      int    `json:"days,omitempty" jsonschema:"days of hourly history to scan, 21 by default; a single item always uses the last ~15 days"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum average daily volume for a scan, 1000 by default"`
	MinPrice  int    `json:"min_price,omitempty" jsonschema:"minimum price in coins for a scan, 100 by default"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
}

type cyclesOut struct {
	Cycles []market.DailyCycle `json:"cycles"`
	// Fetched is how many new hourly windows the archive downloaded.
	Fetched int  `json:"fetched_windows,omitempty"`
	single  bool `json:"-"`
}

func (o cyclesOut) text() string {
	var b strings.Builder
	if o.single && len(o.Cycles) == 1 {
		c := o.Cycles[0]
		fmt.Fprintf(&b, "%s over %d days: cheapest %02d:00 UTC, dearest %02d:00 UTC, amplitude %+.1f%%\n", c.Name, c.Days, c.LowHour, c.HighHour, c.Amplitude*100)
		fmt.Fprintf(&b, "held-out days: high beat low on %d of %d, median edge after tax %+.1f%%\n", c.Wins, c.TestDays, c.Edge*100)
		fmt.Fprintf(&b, "selling clusters at %02d:00-%02d:00 UTC with %s of sell volume (even spread is 12.5%%)\n\n", c.DumpHour, (c.DumpHour+3)%24, pct(c.DumpShare))
		lo, hi := c.Profile[c.LowHour], c.Profile[c.HighHour]
		t := newTable(&b, "HOUR (UTC)", "PRICE VS DAY MEAN")
		for h, v := range c.Profile {
			bar := ""
			if v > 0 && hi > lo {
				bar = strings.Repeat("#", 1+int((v-lo)/(hi-lo)*30))
			}
			t.row(fmt.Sprintf("%02d:00", h), fmt.Sprintf("%+.2f%% %s", (v-1)*100, bar))
		}
		t.flush()
		return b.String()
	}
	if o.Fetched > 0 {
		fmt.Fprintf(&b, "archived %d new hourly windows\n\n", o.Fetched)
	}
	t := newTable(&b, "ITEM", "BUY (UTC)", "SELL (UTC)", "AMPLITUDE", "HELD-OUT WINS", "EDGE AFTER TAX", "DUMP HOURS", "DUMP SHARE", "DAILY VOL")
	for _, c := range o.Cycles {
		t.row(c.Name, fmt.Sprintf("%02d:00", c.LowHour), fmt.Sprintf("%02d:00", c.HighHour), fmt.Sprintf("%+.1f%%", c.Amplitude*100),
			fmt.Sprintf("%d/%d", c.Wins, c.TestDays), fmt.Sprintf("%+.1f%%", c.Edge*100), fmt.Sprintf("%02d-%02d", c.DumpHour, (c.DumpHour+3)%24), pct(c.DumpShare), num(int64(c.Volume)))
	}
	t.flush()
	b.WriteString("\nbuy and sell hours come from the earlier two thirds of the days; wins and edge are measured on the rest.\nedge assumes fills at the hour's average price.\n")
	return b.String()
}

var cyclesCmd = newCommand("cycles", "find hour-of-day price cycles and bot-dump hours, tested on held-out days; profile one item or scan the market",
	func(ctx context.Context, d *deps, in cyclesIn) (cyclesOut, error) {
		cat, err := d.itemCatalog(ctx)
		if err != nil {
			return cyclesOut{}, err
		}
		if in.Item != "" {
			item, err := d.findItem(ctx, in.Item)
			if err != nil {
				return cyclesOut{}, err
			}
			// One request returns about 15 days of hourly points.
			ts, err := d.prices.Timeseries(ctx, item.ID, prices.OneHour)
			if err != nil {
				return cyclesOut{}, err
			}
			got := market.DailyCycles(map[int][]prices.TimeseriesDatapoint{item.ID: ts.Data}, cat, 6)
			if len(got) == 0 {
				return cyclesOut{}, fmt.Errorf("%s trades too rarely to find a daily pattern", item.Name)
			}
			return cyclesOut{Cycles: got, single: true}, nil
		}

		// 21 days leaves 7 held-out days to test each pattern on.
		days := orDefault(in.Days, 21)
		if days < 6 || days > 90 {
			return cyclesOut{}, fmt.Errorf("days must be between 6 and 90")
		}
		a, err := d.priceArchive()
		if err != nil {
			return cyclesOut{}, err
		}
		to := time.Now().UTC().Truncate(24 * time.Hour)
		from := to.Add(-time.Duration(days) * 24 * time.Hour)
		fetched, err := a.Ensure(ctx, prices.OneHour, from, to)
		if err != nil {
			return cyclesOut{}, err
		}
		series, err := a.Series(prices.OneHour, from, to)
		if err != nil {
			return cyclesOut{}, err
		}
		latest, err := d.prices.Latest(ctx)
		if err != nil {
			return cyclesOut{}, err
		}
		var out []market.DailyCycle
		for _, c := range market.DailyCycles(series, cat, 6) {
			price := latest[c.ItemID].Low
			if c.Edge <= 0 || c.Consistency < 0.6 || c.Volume < orDefault(in.MinVolume, 1000) || price < orDefault(in.MinPrice, 100) {
				continue
			}
			out = append(out, c)
		}
		return cyclesOut{Cycles: top(out, orDefault(in.Top, 20)), Fetched: fetched}, nil
	})
