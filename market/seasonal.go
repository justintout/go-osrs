package market

import (
	"math"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Season is an item's typical price pattern across the months of a year.
type Season struct {
	// Years is how many years had at least 9 months of trades.
	Years int `json:"years"`
	// Low and High are the months (1-12) where the price is usually
	// lowest and highest across all years.
	Low  time.Month `json:"low"`
	High time.Month `json:"high"`
	// Amplitude is the typical high-month price over the low-month price,
	// minus one.
	Amplitude float64 `json:"amplitude"`
	// Wins counts years where, with that year left out, the months picked
	// from the other years still had high above low in the left-out year.
	Wins  int `json:"wins"`
	Tests int `json:"tests"`
	// Edge is the median left-out-year gain, less the 2% GE tax.
	Edge float64 `json:"edge"`
	// Profile is each month's typical price relative to the year's mean,
	// January first.
	Profile [12]float64 `json:"profile"`
}

// Seasonal finds month-of-year patterns in one item's daily series. With
// only a few years of history, each year is tested against months picked
// from the others, so a pattern needs to hold year after year to score.
func Seasonal(points []prices.TimeseriesDatapoint) (Season, bool) {
	years := map[int]*[12]float64{}
	sums, counts := map[int]*[12]float64{}, map[int]*[12]int{}
	for _, p := range points {
		mid, _ := hourPrice(prices.Average{AvgHighPrice: p.AvgHighPrice, HighPriceVolume: p.HighPriceVolume, AvgLowPrice: p.AvgLowPrice, LowPriceVolume: p.LowPriceVolume})
		if mid == 0 {
			continue
		}
		t := p.Timestamp.UTC()
		y, m := t.Year(), int(t.Month())-1
		if sums[y] == nil {
			sums[y], counts[y] = &[12]float64{}, &[12]int{}
		}
		sums[y][m] += mid
		counts[y][m]++
	}
	// Each month's mean price relative to its year's mean of monthly means.
	for y, s := range sums {
		var monthly [12]float64
		total, n := 0.0, 0
		for m := range 12 {
			if counts[y][m] >= 10 {
				monthly[m] = s[m] / float64(counts[y][m])
				total += monthly[m]
				n++
			}
		}
		if n < 9 {
			continue
		}
		rel := &[12]float64{}
		for m, v := range monthly {
			if v > 0 {
				rel[m] = v / (total / float64(n))
			}
		}
		years[y] = rel
	}
	if len(years) < 3 {
		return Season{}, false
	}

	profile := func(skip int) [12]float64 {
		var out [12]float64
		for m := range 12 {
			var vals []float64
			for y, rel := range years {
				if y != skip && rel[m] > 0 {
					vals = append(vals, rel[m])
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

	s := Season{Years: len(years), Profile: profile(-1)}
	lo, hi := lowHigh(s.Profile)
	if lo < 0 || lo == hi {
		return Season{}, false
	}
	s.Low, s.High = time.Month(lo+1), time.Month(hi+1)
	s.Amplitude = s.Profile[hi]/s.Profile[lo] - 1

	var gains []float64
	for y, rel := range years {
		l, h := lowHigh(profile(y))
		if l < 0 || rel[l] == 0 || rel[h] == 0 {
			continue
		}
		g := rel[h]/rel[l] - 1
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
