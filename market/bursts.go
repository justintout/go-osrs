package market

import (
	"math"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Burst is one timeseries window where one side traded far more than
// usual at a price far from normal: the mark of a single large order
// rather than a market-wide move.
type Burst struct {
	Time time.Time `json:"time"`
	// Side is "buy" for trades at the instant-buy price and "sell" for
	// trades at the instant-sell price.
	Side   string `json:"side"`
	Price  int    `json:"price"`
	Volume int    `json:"volume"`
	// NormalPrice is the side's median trade price across the series.
	NormalPrice int `json:"normal_price"`
	// VolumeMultiple is Volume over the side's median window volume.
	VolumeMultiple float64 `json:"volume_multiple"`
	// BuyLimitShare is Volume over the item's 4-hour buy limit. Buy-side
	// bursts near or under 1 fit a single account buying its limit.
	BuyLimitShare float64 `json:"buy_limit_share,omitempty"`
}

// Bursts scans a timeseries for windows where one side's volume is at
// least 10 times its median and its price is at least 20% from that
// side's median price. Each side has its own reference because instant
// buys normally trade above instant sells. buyLimit is the item's 4-hour
// limit, or 0.
func Bursts(points []prices.TimeseriesDatapoint, buyLimit int) []Burst {
	var all, highPrices, lowPrices, highVols, lowVols []float64
	for _, p := range points {
		if p.HighPriceVolume > 0 {
			highPrices = append(highPrices, float64(p.AvgHighPrice))
		}
		if p.LowPriceVolume > 0 {
			lowPrices = append(lowPrices, float64(p.AvgLowPrice))
		}
		highVols = append(highVols, float64(p.HighPriceVolume))
		lowVols = append(lowVols, float64(p.LowPriceVolume))
	}
	all = append(append(all, highPrices...), lowPrices...)
	if len(all) == 0 {
		return nil
	}
	// A side with too few trades to have a median of its own falls back
	// to the median of both sides.
	reference := func(side []float64) float64 {
		if len(side) < 5 {
			return median(all)
		}
		return median(side)
	}
	normalHigh, normalLow := reference(highPrices), reference(lowPrices)
	baseHigh, baseLow := math.Max(median(highVols), 1), math.Max(median(lowVols), 1)

	var out []Burst
	check := func(t time.Time, side string, price, volume int, base, normal float64) {
		if volume == 0 || float64(volume) < 10*base || math.Abs(float64(price)/normal-1) < 0.2 {
			return
		}
		b := Burst{Time: t, Side: side, Price: price, Volume: volume, NormalPrice: int(normal), VolumeMultiple: float64(volume) / base}
		if buyLimit > 0 && side == "buy" {
			b.BuyLimitShare = float64(volume) / float64(buyLimit)
		}
		out = append(out, b)
	}
	for _, p := range points {
		check(p.Timestamp, "buy", p.AvgHighPrice, p.HighPriceVolume, baseHigh, normalHigh)
		check(p.Timestamp, "sell", p.AvgLowPrice, p.LowPriceVolume, baseLow, normalLow)
	}
	return out
}
