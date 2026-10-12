// Package market finds unusual Grand Exchange price moves: items moving
// away from their own recent baseline, and items that changed across a
// game update.
package market

import (
	"math"
	"sort"

	"github.com/justintout/go-osrs/prices"
)

// Mover is an item whose latest hourly price departs from its baseline.
type Mover struct {
	ItemID int    `json:"item_id"`
	Name   string `json:"name"`
	// Price is the latest hour's volume-weighted average trade price.
	Price int `json:"price"`
	// Baseline is the median hourly price over the earlier hours.
	Baseline int `json:"baseline"`
	// Change is the fractional move from Baseline, such as -0.12.
	Change float64 `json:"change"`
	// Score is the move in robust standard deviations of the item's own
	// hourly prices, so a volatile item needs a bigger move to score high.
	Score float64 `json:"score"`
	// Volume is the latest hour's trades and VolumeRatio its multiple of
	// the median hourly volume.
	Volume      int     `json:"volume"`
	VolumeRatio float64 `json:"volume_ratio"`
}

// Movers compares the last hour in hours, ordered oldest first, against
// the earlier hours and returns items by the size of their move, largest
// first. Items need at least minVolume trades in the latest hour, a
// baseline of at least minPrice, and trades in at least half of the
// earlier hours.
func Movers(hours []prices.Averages, catalog *prices.Catalog, minVolume, minPrice int) []Mover {
	if len(hours) < 3 {
		return nil
	}
	latest, history := hours[len(hours)-1], hours[:len(hours)-1]
	var out []Mover
	for id, a := range latest {
		price, volume := hourPrice(a)
		if volume < minVolume || price == 0 {
			continue
		}
		var ps, vs []float64
		for _, h := range history {
			if p, v := hourPrice(h[id]); p > 0 {
				ps = append(ps, p)
				vs = append(vs, float64(v))
			}
		}
		if len(ps) < len(history)/2 {
			continue
		}
		base := median(ps)
		if base < float64(minPrice) {
			continue
		}
		// 1.4826 scales MAD to a standard deviation. Prices move in whole
		// coins, so cheap or flat items get a floor of 2% of the baseline;
		// without it a one-coin tick scores as a crash.
		spread := math.Max(1.4826*mad(ps, base), base*0.02)
		m := Mover{
			ItemID: id, Price: int(price), Baseline: int(base),
			Change: price/base - 1, Score: (price - base) / spread,
			Volume: volume, VolumeRatio: float64(volume) / math.Max(median(vs), 1),
		}
		if item, ok := catalog.ByID(id); ok {
			m.Name = item.Name
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return math.Abs(out[i].Score) > math.Abs(out[j].Score) })
	return out
}

// Change is an item's price change between two blocks of hours.
type Change struct {
	ItemID int     `json:"item_id"`
	Name   string  `json:"name"`
	Before int     `json:"before"`
	After  int     `json:"after"`
	Change float64 `json:"change"`
	// Volume is the trades in the after block.
	Volume int `json:"volume"`
}

// Compare returns each item's change from the before hours to the after
// hours, using volume-weighted average prices across each block, largest
// move first. Items need at least minVolume trades in each block and a
// before price of at least minPrice.
func Compare(before, after []prices.Averages, catalog *prices.Catalog, minVolume, minPrice int) []Change {
	b, bv := blockPrices(before)
	a, av := blockPrices(after)
	var out []Change
	for id, ap := range a {
		bp := b[id]
		if bp < float64(max(minPrice, 1)) || av[id] < minVolume || bv[id] < minVolume {
			continue
		}
		c := Change{ItemID: id, Before: int(bp), After: int(ap), Change: ap/bp - 1, Volume: av[id]}
		if item, ok := catalog.ByID(id); ok {
			c.Name = item.Name
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return math.Abs(out[i].Change) > math.Abs(out[j].Change) })
	return out
}

// hourPrice is the volume-weighted average of an hour's high and low
// trade prices, and the hour's total volume.
func hourPrice(a prices.Average) (float64, int) {
	v := a.HighPriceVolume + a.LowPriceVolume
	if v == 0 {
		return 0, 0
	}
	return (float64(a.AvgHighPrice)*float64(a.HighPriceVolume) + float64(a.AvgLowPrice)*float64(a.LowPriceVolume)) / float64(v), v
}

func blockPrices(hours []prices.Averages) (map[int]float64, map[int]int) {
	value, volume := map[int]float64{}, map[int]int{}
	for _, h := range hours {
		for id, a := range h {
			p, v := hourPrice(a)
			value[id] += p * float64(v)
			volume[id] += v
		}
	}
	for id := range value {
		value[id] /= float64(volume[id])
	}
	return value, volume
}

func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func mad(xs []float64, center float64) float64 {
	dev := make([]float64, len(xs))
	for i, x := range xs {
		dev[i] = math.Abs(x - center)
	}
	return median(dev)
}
