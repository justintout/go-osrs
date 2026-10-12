package market

import (
	"math"
	"sort"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Season is an item's typical price pattern across the months of a year.
type Season struct {
	// Years is how many years contributed detrended months.
	Years int `json:"years"`
	// Low and High are the months where the price usually sits furthest
	// below and above its trend.
	Low  time.Month `json:"low"`
	High time.Month `json:"high"`
	// Amplitude is the typical high-month price over the low-month price,
	// minus one, after removing the trend.
	Amplitude float64 `json:"amplitude"`
	// Wins counts years where, with that year left out, the months picked
	// from the other years still had high above low in the left-out year.
	Wins  int `json:"wins"`
	Tests int `json:"tests"`
	// Edge is the median left-out-year gain, less the 2% GE tax.
	Edge float64 `json:"edge"`
	// Profile is each month's typical price relative to its trend,
	// January first.
	Profile [12]float64 `json:"profile"`
}

// Seasonal finds month-of-year patterns in one item's daily series.
//
// Each month's mean price is divided by a centered 12-month moving
// average, so a long rise or fall does not pass for a season; a price that
// drifts down all year would otherwise always look dearest in January.
// With only a few years of history, each year is tested against months
// picked from the others.
func Seasonal(points []prices.TimeseriesDatapoint) (Season, bool) {
	type monthKey struct{ year, month int }
	sums, counts := map[monthKey]float64{}, map[monthKey]int{}
	for _, p := range points {
		mid, _ := hourPrice(prices.Average{AvgHighPrice: p.AvgHighPrice, HighPriceVolume: p.HighPriceVolume, AvgLowPrice: p.AvgLowPrice, LowPriceVolume: p.LowPriceVolume})
		if mid == 0 {
			continue
		}
		t := p.Timestamp.UTC()
		k := monthKey{t.Year(), int(t.Month())}
		sums[k] += mid
		counts[k]++
	}
	var keys []monthKey
	for k, n := range counts {
		if n >= 10 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].year*12+keys[i].month < keys[j].year*12+keys[j].month
	})
	// Index months on one continuous axis so gaps are visible.
	monthly := map[int]float64{}
	for _, k := range keys {
		monthly[k.year*12+k.month-1] = sums[k] / float64(counts[k])
	}

	// rel[year][month] is the month's price over its 2x12 centered moving
	// average, which needs every month from 6 before to 6 after.
	rel := map[int]*[12]float64{}
	for idx, price := range monthly {
		total, ok := 0.0, true
		for off := -6; off <= 6; off++ {
			v, have := monthly[idx+off]
			if !have {
				ok = false
				break
			}
			w := 1.0
			if off == -6 || off == 6 {
				w = 0.5
			}
			total += w * v
		}
		if !ok {
			continue
		}
		year, month := idx/12, idx%12
		if rel[year] == nil {
			rel[year] = &[12]float64{}
		}
		rel[year][month] = price / (total / 12)
	}
	if len(rel) < 2 {
		return Season{}, false
	}

	profile := func(skip int) [12]float64 {
		var out [12]float64
		for m := range 12 {
			var vals []float64
			for y, r := range rel {
				if y != skip && r[m] > 0 {
					vals = append(vals, r[m])
				}
			}
			if len(vals) > 0 {
				out[m] = median(vals)
			}
		}
		return out
	}
	lowHigh := func(p [12]float64) (int, int) {
		lo, hi := -1, -1
		for m, v := range p {
			if v == 0 {
				continue
			}
			if lo < 0 || v < p[lo] {
				lo = m
			}
			if hi < 0 || v > p[hi] {
				hi = m
			}
		}
		return lo, hi
	}

	s := Season{Years: len(rel), Profile: profile(-1)}
	lo, hi := lowHigh(s.Profile)
	if lo < 0 || lo == hi {
		return Season{}, false
	}
	s.Low, s.High = time.Month(lo+1), time.Month(hi+1)
	s.Amplitude = s.Profile[hi]/s.Profile[lo] - 1

	var gains []float64
	for y, r := range rel {
		l, h := lowHigh(profile(y))
		if l < 0 || r[l] == 0 || r[h] == 0 {
			continue
		}
		g := r[h]/r[l] - 1
		gains = append(gains, g)
		if g > 0 {
			s.Wins++
		}
	}
	if len(gains) == 0 {
		return Season{}, false
	}
	s.Tests = len(gains)
	s.Edge = median(gains) - 0.02
	return s, !math.IsNaN(s.Edge)
}
