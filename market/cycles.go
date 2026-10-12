package market

import (
	"math"
	"sort"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// DailyCycle is an item's typical price pattern across the UTC day.
type DailyCycle struct {
	ItemID int    `json:"item_id"`
	Name   string `json:"name"`
	// Days is how many days had enough trades to use.
	Days int `json:"days"`
	// LowHour and HighHour are the UTC hours where the price is usually
	// lowest and highest, found on the earlier two thirds of the days.
	LowHour  int `json:"low_hour"`
	HighHour int `json:"high_hour"`
	// Amplitude is the typical high-hour price over the low-hour price,
	// minus one, on those earlier days.
	Amplitude float64 `json:"amplitude"`
	// Consistency is the share of the later, held-out days on which the
	// high hour beat the low hour: Wins of TestDays.
	Consistency float64 `json:"consistency"`
	TestDays    int     `json:"test_days"`
	Wins        int     `json:"wins"`
	// Edge is the median held-out gain from buying at the low hour and
	// selling at the high hour, less the 2% GE tax. It assumes trades
	// fill at the hour's average price.
	Edge float64 `json:"edge"`
	// DumpHour starts the 3-hour block with the most instant-sell volume,
	// and DumpShare is that block's share of sell volume. A share well
	// above 0.125 (3 of 24 hours) means selling clusters at that time,
	// often a sign of bots.
	DumpHour  int     `json:"dump_hour"`
	DumpShare float64 `json:"dump_share"`
	// Volume is the average daily volume.
	Volume int `json:"volume"`
	// Profile is each hour's typical price relative to the day's mean.
	Profile [24]float64 `json:"profile"`
}

// DailyCycles finds hour-of-day patterns in hourly series keyed by item
// ID. A day counts if at least 18 of its hours traded. Items need at
// least minDays such days, and the pattern is tested on the last third of
// them, at least 4 days, so an hour that only looked good by chance shows
// low Consistency.
func DailyCycles(series map[int][]prices.TimeseriesDatapoint, catalog *prices.Catalog, minDays int) []DailyCycle {
	minDays = max(minDays, 6)
	var out []DailyCycle
	for id, points := range series {
		c, ok := dailyCycle(points, minDays)
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

type day struct {
	low, high [24]float64 // side prices; 0 where that side did not trade
	rel       [24]float64 // relative price, 0 where neither side traded
	sellVol   [24]int
	volume    int
}

// relative fills d.rel: each side's price over that side's mean for the
// day, averaged across the sides that traded. Normalizing sides apart
// keeps an hour that happened to see mostly instant sells from looking
// cheap only because of the bid-ask spread. It reports whether the day
// has enough trading to use.
func (d *day) relative() bool {
	mean := func(side [24]float64) (float64, int) {
		sum, n := 0.0, 0
		for _, p := range side {
			if p > 0 {
				sum += p
				n++
			}
		}
		if n == 0 {
			return 0, 0
		}
		return sum / float64(n), n
	}
	lowMean, lowN := mean(d.low)
	highMean, highN := mean(d.high)
	if max(lowN, highN) < 18 {
		return false
	}
	for h := range 24 {
		sum, n := 0.0, 0
		if d.low[h] > 0 && lowN >= 12 {
			sum += d.low[h] / lowMean
			n++
		}
		if d.high[h] > 0 && highN >= 12 {
			sum += d.high[h] / highMean
			n++
		}
		if n > 0 {
			d.rel[h] = sum / float64(n)
		}
	}
	return true
}

func dailyCycle(points []prices.TimeseriesDatapoint, minDays int) (DailyCycle, bool) {
	byDay := map[int64]*day{}
	for _, p := range points {
		t := p.Timestamp.UTC()
		key := t.Truncate(24 * time.Hour).Unix()
		d := byDay[key]
		if d == nil {
			d = &day{}
			byDay[key] = d
		}
		if p.LowPriceVolume > 0 {
			d.low[t.Hour()] = float64(p.AvgLowPrice)
		}
		if p.HighPriceVolume > 0 {
			d.high[t.Hour()] = float64(p.AvgHighPrice)
		}
		d.sellVol[t.Hour()] += p.LowPriceVolume
		d.volume += p.LowPriceVolume + p.HighPriceVolume
	}

	var keys []int64
	for k, d := range byDay {
		if d.relative() {
			keys = append(keys, k)
		}
	}
	if len(keys) < minDays {
		return DailyCycle{}, false
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	split := len(keys) * 2 / 3
	train, test := keys[:split], keys[split:]
	if len(test) < 4 {
		return DailyCycle{}, false
	}

	var c DailyCycle
	c.Days = len(keys)
	for h := range 24 {
		var vals []float64
		for _, k := range train {
			if r := byDay[k].rel[h]; r > 0 {
				vals = append(vals, r)
			}
		}
		if len(vals) > 0 {
			c.Profile[h] = median(vals)
		}
	}
	for h, v := range c.Profile {
		if v > 0 && (c.Profile[c.LowHour] == 0 || v < c.Profile[c.LowHour]) {
			c.LowHour = h
		}
		if v > c.Profile[c.HighHour] {
			c.HighHour = h
		}
	}
	if c.LowHour == c.HighHour || c.Profile[c.LowHour] == 0 {
		return DailyCycle{}, false
	}
	c.Amplitude = c.Profile[c.HighHour]/c.Profile[c.LowHour] - 1

	var gains []float64
	wins := 0
	for _, k := range test {
		lo, hi := byDay[k].rel[c.LowHour], byDay[k].rel[c.HighHour]
		if lo == 0 || hi == 0 {
			continue
		}
		g := hi/lo - 1
		gains = append(gains, g)
		if g > 0 {
			wins++
		}
	}
	if len(gains) < 4 {
		return DailyCycle{}, false
	}
	c.TestDays, c.Wins = len(gains), wins
	c.Consistency = float64(wins) / float64(len(gains))
	c.Edge = median(gains) - 0.02

	var sell [24]int
	total, volume := 0, 0
	for _, k := range keys {
		for h, v := range byDay[k].sellVol {
			sell[h] += v
			total += v
		}
		volume += byDay[k].volume
	}
	c.Volume = volume / len(keys)
	if total > 0 {
		best := -1
		for h := range 24 {
			block := sell[h] + sell[(h+1)%24] + sell[(h+2)%24]
			if block > best {
				best, c.DumpHour = block, h
			}
		}
		c.DumpShare = float64(best) / float64(total)
	}
	return c, !math.IsNaN(c.Edge)
}
