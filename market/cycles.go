package market

import (
	"math"
	"sort"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Cycle is an item's typical price pattern within a day or a week.
type Cycle struct {
	ItemID int    `json:"item_id"`
	Name   string `json:"name"`
	// Period is "daily" (buckets are UTC hours, 0-23) or "weekly"
	// (buckets are weekdays, 0 is Sunday).
	Period string `json:"period"`
	// Groups is how many days or weeks had enough trades to use.
	Groups int `json:"groups"`
	// Low and High are the buckets where the price is usually lowest and
	// highest, found on the earlier two thirds of the groups.
	Low  int `json:"low"`
	High int `json:"high"`
	// Amplitude is the typical high-bucket price over the low-bucket
	// price, minus one, on those earlier groups.
	Amplitude float64 `json:"amplitude"`
	// Wins of TestDays held-out groups had the high bucket beat the low
	// bucket; Consistency is their ratio.
	Consistency float64 `json:"consistency"`
	TestGroups  int     `json:"test_groups"`
	Wins        int     `json:"wins"`
	// Edge is the median held-out gain from buying in the low bucket and
	// selling in the high one, less the 2% GE tax. It assumes trades fill
	// at the bucket's average price.
	Edge float64 `json:"edge"`
	// Dump starts the block with the most instant-sell volume (3 hours,
	// or 1 day for weekly) and DumpShare is that block's share. Selling
	// that bunches far above an even spread often comes from bots.
	Dump      int     `json:"dump"`
	DumpShare float64 `json:"dump_share"`
	// Volume is the average volume per bucket times the bucket count: a
	// day's or a week's typical volume.
	Volume int `json:"volume"`
	// Profile is each bucket's typical price relative to the group mean.
	Profile []float64 `json:"profile"`
}

type period struct {
	name      string
	buckets   int
	group     func(time.Time) int64
	bucket    func(time.Time) int
	minFilled int // buckets that must trade for a group to count
	sideMin   int // buckets a side must trade to be normalized
	dumpSpan  int
}

var (
	daily = period{
		name: "daily", buckets: 24, minFilled: 18, sideMin: 12, dumpSpan: 3,
		group:  func(t time.Time) int64 { return t.Truncate(24 * time.Hour).Unix() },
		bucket: func(t time.Time) int { return t.Hour() },
	}
	weekly = period{
		name: "weekly", buckets: 7, minFilled: 6, sideMin: 5, dumpSpan: 1,
		group: func(t time.Time) int64 {
			d := t.Truncate(24 * time.Hour)
			return d.AddDate(0, 0, -int(d.Weekday())).Unix()
		},
		bucket: func(t time.Time) int { return int(t.Weekday()) },
	}
)

// DailyCycles finds hour-of-day patterns in hourly series keyed by item
// ID. A day counts if at least 18 of its hours traded. Items need at
// least minDays such days, and patterns are tested on the last third of
// them, at least 4 days, so an hour that only looked good by chance shows
// low Consistency.
func DailyCycles(series map[int][]prices.TimeseriesDatapoint, catalog *prices.Catalog, minDays int) []Cycle {
	return cycles(series, catalog, daily, max(minDays, 6), 4)
}

// WeeklyCycles finds day-of-week patterns in daily series keyed by item
// ID. A week counts if at least 6 of its days traded. Items need at least
// minWeeks such weeks, and patterns are tested on the last third of them,
// at least 3 weeks.
func WeeklyCycles(series map[int][]prices.TimeseriesDatapoint, catalog *prices.Catalog, minWeeks int) []Cycle {
	return cycles(series, catalog, weekly, max(minWeeks, 6), 3)
}

func cycles(series map[int][]prices.TimeseriesDatapoint, catalog *prices.Catalog, p period, minGroups, minTest int) []Cycle {
	var out []Cycle
	for id, points := range series {
		c, ok := findCycle(points, p, minGroups, minTest)
		if !ok {
			continue
		}
		c.ItemID = id
		if item, ok := catalog.ByID(id); ok {
			c.Name = item.Name
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Edge*out[i].Consistency > out[j].Edge*out[j].Consistency })
	return out
}

type group struct {
	low, high []float64 // side prices by bucket; 0 where that side did not trade
	rel       []float64 // relative price by bucket; 0 where neither traded
	sellVol   []int
	volume    int
}

// relative fills g.rel: each side's price over that side's mean for the
// group, averaged across the sides that traded. Normalizing sides apart
// keeps a bucket that happened to see mostly instant sells from looking
// cheap only because of the bid-ask spread. It reports whether the group
// has enough trading to use.
func (g *group) relative(p period) bool {
	mean := func(side []float64) (float64, int) {
		sum, n := 0.0, 0
		for _, v := range side {
			if v > 0 {
				sum += v
				n++
			}
		}
		if n == 0 {
			return 0, 0
		}
		return sum / float64(n), n
	}
	lowMean, lowN := mean(g.low)
	highMean, highN := mean(g.high)
	if max(lowN, highN) < p.minFilled {
		return false
	}
	for b := range p.buckets {
		sum, n := 0.0, 0
		if g.low[b] > 0 && lowN >= p.sideMin {
			sum += g.low[b] / lowMean
			n++
		}
		if g.high[b] > 0 && highN >= p.sideMin {
			sum += g.high[b] / highMean
			n++
		}
		if n > 0 {
			g.rel[b] = sum / float64(n)
		}
	}
	return true
}

func findCycle(points []prices.TimeseriesDatapoint, p period, minGroups, minTest int) (Cycle, bool) {
	groups := map[int64]*group{}
	for _, pt := range points {
		t := pt.Timestamp.UTC()
		key := p.group(t)
		g := groups[key]
		if g == nil {
			g = &group{low: make([]float64, p.buckets), high: make([]float64, p.buckets), rel: make([]float64, p.buckets), sellVol: make([]int, p.buckets)}
			groups[key] = g
		}
		b := p.bucket(t)
		if pt.LowPriceVolume > 0 {
			g.low[b] = float64(pt.AvgLowPrice)
		}
		if pt.HighPriceVolume > 0 {
			g.high[b] = float64(pt.AvgHighPrice)
		}
		g.sellVol[b] += pt.LowPriceVolume
		g.volume += pt.LowPriceVolume + pt.HighPriceVolume
	}

	var keys []int64
	for k, g := range groups {
		if g.relative(p) {
			keys = append(keys, k)
		}
	}
	if len(keys) < minGroups {
		return Cycle{}, false
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	split := len(keys) * 2 / 3
	train, test := keys[:split], keys[split:]
	if len(test) < minTest {
		return Cycle{}, false
	}

	c := Cycle{Period: p.name, Groups: len(keys), Profile: make([]float64, p.buckets)}
	for b := range p.buckets {
		var vals []float64
		for _, k := range train {
			if r := groups[k].rel[b]; r > 0 {
				vals = append(vals, r)
			}
		}
		if len(vals) > 0 {
			c.Profile[b] = median(vals)
		}
	}
	for b, v := range c.Profile {
		if v > 0 && (c.Profile[c.Low] == 0 || v < c.Profile[c.Low]) {
			c.Low = b
		}
		if v > c.Profile[c.High] {
			c.High = b
		}
	}
	if c.Low == c.High || c.Profile[c.Low] == 0 {
		return Cycle{}, false
	}
	c.Amplitude = c.Profile[c.High]/c.Profile[c.Low] - 1

	var gains []float64
	for _, k := range test {
		lo, hi := groups[k].rel[c.Low], groups[k].rel[c.High]
		if lo == 0 || hi == 0 {
			continue
		}
		g := hi/lo - 1
		gains = append(gains, g)
		if g > 0 {
			c.Wins++
		}
	}
	if len(gains) < minTest {
		return Cycle{}, false
	}
	c.TestGroups = len(gains)
	c.Consistency = float64(c.Wins) / float64(len(gains))
	c.Edge = median(gains) - 0.02

	sell := make([]int, p.buckets)
	total, volume := 0, 0
	for _, k := range keys {
		for b, v := range groups[k].sellVol {
			sell[b] += v
			total += v
		}
		volume += groups[k].volume
	}
	c.Volume = volume / len(keys)
	if total > 0 {
		best := -1
		for b := range p.buckets {
			block := 0
			for s := range p.dumpSpan {
				block += sell[(b+s)%p.buckets]
			}
			if block > best {
				best, c.Dump = block, b
			}
		}
		c.DumpShare = float64(best) / float64(total)
	}
	return c, !math.IsNaN(c.Edge)
}
