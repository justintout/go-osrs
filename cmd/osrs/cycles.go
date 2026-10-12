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
	Period    string `json:"period,omitempty" jsonschema:"daily (default) for hour-of-day cycles or weekly for day-of-week cycles"`
	Days      int    `json:"days,omitempty" jsonschema:"days of history for a market scan: 21 by default for daily, 182 for weekly"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum typical volume per day (daily) or week (weekly) for a scan, 1000 by default"`
	MinPrice  int    `json:"min_price,omitempty" jsonschema:"minimum price in coins for a scan, 100 by default"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
}

type cyclesOut struct {
	Cycles []market.Cycle `json:"cycles"`
	// Fetched is how many new windows the archive downloaded.
	Fetched int  `json:"fetched_windows,omitempty"`
	single  bool `json:"-"`
}

func bucketName(period string, b int) string {
	if period == "weekly" {
		return time.Weekday(b).String()[:3]
	}
	return fmt.Sprintf("%02d:00", b)
}

func (o cyclesOut) text() string {
	var b strings.Builder
	if o.single && len(o.Cycles) == 1 {
		c := o.Cycles[0]
		unit, groups := " UTC", "days"
		if c.Period == "weekly" {
			unit, groups = "", "weeks"
		}
		fmt.Fprintf(&b, "%s over %d %s: cheapest %s%s, dearest %s%s, amplitude %+.1f%%\n", c.Name, c.Groups, groups,
			bucketName(c.Period, c.Low), unit, bucketName(c.Period, c.High), unit, c.Amplitude*100)
		fmt.Fprintf(&b, "held-out %s: high beat low on %d of %d, median edge after tax %+.1f%%\n", groups, c.Wins, c.TestGroups, c.Edge*100)
		fmt.Fprintf(&b, "selling bunches from %s with %s of sell volume\n\n", bucketName(c.Period, c.Dump), pct(c.DumpShare))
		lo, hi := c.Profile[c.Low], c.Profile[c.High]
		t := newTable(&b, "WHEN", "PRICE VS MEAN")
		for i, v := range c.Profile {
			bar := ""
			if v > 0 && hi > lo {
				bar = strings.Repeat("#", 1+int((v-lo)/(hi-lo)*30))
			}
			t.row(bucketName(c.Period, i), fmt.Sprintf("%+.2f%% %s", (v-1)*100, bar))
		}
		t.flush()
		return b.String()
	}
	if o.Fetched > 0 {
		fmt.Fprintf(&b, "archived %d new windows\n\n", o.Fetched)
	}
	t := newTable(&b, "ITEM", "BUY", "SELL", "AMPLITUDE", "HELD-OUT WINS", "EDGE AFTER TAX", "DUMP FROM", "DUMP SHARE", "VOLUME")
	for _, c := range o.Cycles {
		t.row(c.Name, bucketName(c.Period, c.Low), bucketName(c.Period, c.High), fmt.Sprintf("%+.1f%%", c.Amplitude*100),
			fmt.Sprintf("%d/%d", c.Wins, c.TestGroups), fmt.Sprintf("%+.1f%%", c.Edge*100), bucketName(c.Period, c.Dump), pct(c.DumpShare), num(int64(c.Volume)))
	}
	t.flush()
	b.WriteString("\nbuy and sell times come from the earlier two thirds of the history; wins and edge are measured on the rest.\nedge assumes fills at the average price; run backtest before trusting a daily cycle.\n")
	return b.String()
}

var cyclesCmd = newCommand("cycles", "find hour-of-day or day-of-week price cycles and dump times, tested on held-out history; profile one item or scan the market",
	func(ctx context.Context, d *deps, in cyclesIn) (cyclesOut, error) {
		weekly := false
		switch in.Period {
		case "", "daily":
		case "weekly":
			weekly = true
		default:
			return cyclesOut{}, fmt.Errorf("period must be daily or weekly")
		}
		step, find, minGroups := prices.OneHour, market.DailyCycles, 6
		if weekly {
			step, find = prices.TwentyFourHour, market.WeeklyCycles
		}
		cat, err := d.itemCatalog(ctx)
		if err != nil {
			return cyclesOut{}, err
		}
		if in.Item != "" {
			item, err := d.findItem(ctx, in.Item)
			if err != nil {
				return cyclesOut{}, err
			}
			// One request returns 365 windows: about 15 days hourly or a
			// year daily.
			ts, err := d.prices.Timeseries(ctx, item.ID, step)
			if err != nil {
				return cyclesOut{}, err
			}
			got := find(map[int][]prices.TimeseriesDatapoint{item.ID: ts.Data}, cat, minGroups)
			if len(got) == 0 {
				return cyclesOut{}, fmt.Errorf("%s trades too rarely to find a pattern", item.Name)
			}
			return cyclesOut{Cycles: got, single: true}, nil
		}

		days := orDefault(in.Days, 21)
		if weekly {
			days = orDefault(in.Days, 182)
		}
		if days < 6 || days > 730 {
			return cyclesOut{}, fmt.Errorf("days must be between 6 and 730")
		}
		a, err := d.priceArchive()
		if err != nil {
			return cyclesOut{}, err
		}
		to := time.Now().UTC().Truncate(24 * time.Hour)
		from := to.Add(-time.Duration(days) * 24 * time.Hour)
		fetched, err := a.Ensure(ctx, step, from, to)
		if err != nil {
			return cyclesOut{}, err
		}
		series, err := a.Series(step, from, to)
		if err != nil {
			return cyclesOut{}, err
		}
		latest, err := d.prices.Latest(ctx)
		if err != nil {
			return cyclesOut{}, err
		}
		var out []market.Cycle
		for _, c := range find(series, cat, minGroups) {
			if c.Edge <= 0 || c.Consistency < 0.6 || c.Volume < orDefault(in.MinVolume, 1000) || latest[c.ItemID].Low < orDefault(in.MinPrice, 100) {
				continue
			}
			out = append(out, c)
		}
		return cyclesOut{Cycles: top(out, orDefault(in.Top, 20)), Fetched: fetched}, nil
	})

type seasonalIn struct {
	Item  string `json:"item" arg:"" jsonschema:"item name"`
	Years int    `json:"years,omitempty" jsonschema:"years of daily history, 3 by default and at most 5 (data starts March 2021); the first run downloads one window per missing day"`
}

type seasonalOut struct {
	Item    string        `json:"item"`
	Season  market.Season `json:"season"`
	Fetched int           `json:"fetched_windows,omitempty"`
}

func (o seasonalOut) text() string {
	var b strings.Builder
	s := o.Season
	if o.Fetched > 0 {
		fmt.Fprintf(&b, "archived %d new daily windows\n\n", o.Fetched)
	}
	fmt.Fprintf(&b, "%s over %d years: cheapest in %s, dearest in %s, amplitude %+.1f%%\n", o.Item, s.Years, s.Low, s.High, s.Amplitude*100)
	fmt.Fprintf(&b, "leaving each year out: high beat low in %d of %d years, median edge after tax %+.1f%%\n\n", s.Wins, s.Tests, s.Edge*100)
	lo, hi := s.Profile[s.Low-1], s.Profile[s.High-1]
	t := newTable(&b, "MONTH", "PRICE VS YEAR MEAN")
	for m, v := range s.Profile {
		bar := ""
		if v > 0 && hi > lo {
			bar = strings.Repeat("#", 1+int((v-lo)/(hi-lo)*30))
		}
		t.row(time.Month(m + 1).String()[:3], fmt.Sprintf("%+.1f%% %s", (v-1)*100, bar))
	}
	t.flush()
	return b.String()
}

var seasonalCmd = newCommand("seasonal", "find an item's month-of-year price pattern across years, tested by leaving each year out",
	func(ctx context.Context, d *deps, in seasonalIn) (seasonalOut, error) {
		if err := need("item", in.Item); err != nil {
			return seasonalOut{}, err
		}
		years := orDefault(in.Years, 3)
		if years < 3 || years > 5 {
			return seasonalOut{}, fmt.Errorf("years must be between 3 and 5")
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return seasonalOut{}, err
		}
		a, err := d.priceArchive()
		if err != nil {
			return seasonalOut{}, err
		}
		to := time.Now().UTC().Truncate(24 * time.Hour)
		from := to.AddDate(-years, 0, 0)
		fetched, err := a.Ensure(ctx, prices.TwentyFourHour, from, to)
		if err != nil {
			return seasonalOut{}, err
		}
		series, err := a.Series(prices.TwentyFourHour, from, to)
		if err != nil {
			return seasonalOut{}, err
		}
		s, ok := market.Seasonal(series[item.ID])
		if !ok {
			return seasonalOut{}, fmt.Errorf("%s does not have 3 years with enough trading to compare", item.Name)
		}
		return seasonalOut{Item: item.Name, Season: s, Fetched: fetched}, nil
	})
