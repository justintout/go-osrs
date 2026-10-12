package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/justintout/go-osrs/market"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

type moversIn struct {
	Hours     int    `json:"hours,omitempty" jsonschema:"baseline window in hours before the latest hour, 24 by default"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum trades in the latest hour, 100 by default"`
	MinPrice  int    `json:"min_price,omitempty" jsonschema:"minimum baseline price in coins, 100 by default"`
	Direction string `json:"direction,omitempty" jsonschema:"up, down, or both (default)"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
}

type moversOut struct {
	Hour   time.Time      `json:"hour"`
	Movers []market.Mover `json:"movers"`
}

func (o moversOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "hour starting %s UTC vs the hours before it\n\n", o.Hour.UTC().Format("2006-01-02 15:04"))
	t := newTable(&b, "ITEM", "PRICE", "BASELINE", "CHANGE", "SCORE", "VOLUME", "VOL X")
	for _, m := range o.Movers {
		t.row(m.Name, num(int64(m.Price)), num(int64(m.Baseline)), fmt.Sprintf("%+.1f%%", m.Change*100), fmt.Sprintf("%+.1f", m.Score), num(int64(m.Volume)), fmt.Sprintf("%.1f", m.VolumeRatio))
	}
	t.flush()
	b.WriteString("\nscore is the move in robust standard deviations of the item's own hourly prices\n")
	return b.String()
}

var moversCmd = newCommand("movers", "find items whose price in the last hour broke from their own recent baseline: crashes, spikes, and volume surges",
	func(ctx context.Context, d *deps, in moversIn) (moversOut, error) {
		window := orDefault(in.Hours, 24)
		if window < 3 || window > 168 {
			return moversOut{}, fmt.Errorf("hours must be between 3 and 168")
		}
		// The hour before the current one is the latest complete hour.
		last := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
		var starts []time.Time
		for i := window; i >= 0; i-- {
			starts = append(starts, last.Add(-time.Duration(i)*time.Hour))
		}
		hours, err := fetchHours(ctx, d, starts)
		if err != nil {
			return moversOut{}, err
		}
		cat, err := d.itemCatalog(ctx)
		if err != nil {
			return moversOut{}, err
		}
		var out []market.Mover
		for _, m := range market.Movers(hours, cat, orDefault(in.MinVolume, 100), orDefault(in.MinPrice, 100)) {
			if (in.Direction == "up" && m.Change < 0) || (in.Direction == "down" && m.Change > 0) {
				continue
			}
			out = append(out, m)
		}
		return moversOut{Hour: last, Movers: top(out, orDefault(in.Top, 20))}, nil
	})

type updateImpactIn struct {
	Update    string `json:"update,omitempty" arg:"" jsonschema:"part of a game update's title; the latest game update by default"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum trades in each 6-hour block, 200 by default"`
	MinPrice  int    `json:"min_price,omitempty" jsonschema:"minimum price before the update in coins, 100 by default"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
}

type updateImpactOut struct {
	Update  wiki.Update     `json:"update"`
	Before  time.Time       `json:"before_block_start"`
	After   time.Time       `json:"after_block_start"`
	Changes []market.Change `json:"changes"`
}

func (o updateImpactOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)\nprices %s-%s UTC vs %s-%s UTC\n\n", o.Update.Title, o.Update.Date.Format(time.DateOnly),
		o.Before.Format("Jan 2 15:04"), o.Before.Add(6*time.Hour).Format("15:04"), o.After.Format("Jan 2 15:04"), o.After.Add(6*time.Hour).Format("15:04"))
	t := newTable(&b, "ITEM", "BEFORE", "AFTER", "CHANGE", "VOLUME AFTER")
	for _, c := range o.Changes {
		t.row(c.Name, num(int64(c.Before)), num(int64(c.After)), fmt.Sprintf("%+.1f%%", c.Change*100), num(int64(c.Volume)))
	}
	t.flush()
	return b.String()
}

var updateImpactCmd = newCommand("update-impact", "show which items' prices moved most across a game update, comparing the same hours the day before and the day after",
	func(ctx context.Context, d *deps, in updateImpactIn) (updateImpactOut, error) {
		updates, err := d.wiki.Updates(ctx)
		if err != nil {
			return updateImpactOut{}, err
		}
		var u *wiki.Update
		for i, up := range updates {
			if up.Type != "Game updates" {
				continue
			}
			if in.Update == "" || strings.Contains(strings.ToLower(up.Title), strings.ToLower(in.Update)) {
				u = &updates[i]
				break
			}
		}
		if u == nil {
			return updateImpactOut{}, fmt.Errorf("no game update matching %q", in.Update)
		}
		// Same evening hours on the day before and the day after, so daily
		// trading cycles cancel out.
		day := u.Date.Truncate(24 * time.Hour)
		before, after := day.Add(-6*time.Hour), day.Add(42*time.Hour)
		if after.Add(6 * time.Hour).After(time.Now()) {
			return updateImpactOut{}, fmt.Errorf("%s was too recent; the day after has not finished", u.Title)
		}
		var starts []time.Time
		for i := range 6 {
			starts = append(starts, before.Add(time.Duration(i)*time.Hour))
		}
		for i := range 6 {
			starts = append(starts, after.Add(time.Duration(i)*time.Hour))
		}
		hours, err := fetchHours(ctx, d, starts)
		if err != nil {
			return updateImpactOut{}, err
		}
		cat, err := d.itemCatalog(ctx)
		if err != nil {
			return updateImpactOut{}, err
		}
		changes := market.Compare(hours[:6], hours[6:], cat, orDefault(in.MinVolume, 200), orDefault(in.MinPrice, 100))
		return updateImpactOut{Update: *u, Before: before, After: after, Changes: top(changes, orDefault(in.Top, 20))}, nil
	})

// fetchHours loads hourly price snapshots for each start time, four at a
// time, in the order given.
func fetchHours(ctx context.Context, d *deps, starts []time.Time) ([]prices.Averages, error) {
	out := make([]prices.Averages, len(starts))
	errs := make([]error, len(starts))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, s := range starts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i], errs[i] = d.prices.OneHourStartingAt(ctx, s)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
