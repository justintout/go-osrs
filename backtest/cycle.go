package backtest

import (
	"sort"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Cycle buys once a day at BuyHour and sells at SellHour, both UTC. Its
// limit prices are the median price at those hours over the last
// Lookback days: the instant-sell price for buying and the instant-buy
// price for selling. A position not sold within MaxHold is sold at
// market.
type Cycle struct {
	BuyHour  int
	SellHour int
	// Quantity is the most to buy per day.
	Quantity int
	// Lookback is days of history for limit prices, 3 by default.
	Lookback int
	// MaxHold is 24 hours by default.
	MaxHold time.Duration
}

// Orders implements Strategy.
func (c Cycle) Orders(i int, history []prices.TimeseriesDatapoint, pos Position) []Order {
	if i == 0 {
		return nil
	}
	lookback, maxHold := c.Lookback, c.MaxHold
	if lookback <= 0 {
		lookback = 3
	}
	if maxHold <= 0 {
		maxHold = 24 * time.Hour
	}
	// The window being traded is one step after the last seen window.
	step := time.Hour
	if i >= 2 {
		step = history[i-1].Timestamp.Sub(history[i-2].Timestamp)
	}
	now := history[i-1].Timestamp.Add(step).UTC()

	if pos.Units > 0 && now.Sub(pos.Since) >= maxHold {
		return []Order{{Side: Sell, Quantity: pos.Units, Market: true}}
	}
	switch {
	case now.Hour() == c.BuyHour && pos.Units == 0:
		if p := medianAt(history, c.BuyHour, lookback, func(d prices.TimeseriesDatapoint) (int, bool) {
			return d.AvgLowPrice, d.LowPriceVolume > 0
		}); p > 0 {
			return []Order{{Side: Buy, Price: p, Quantity: c.Quantity}}
		}
	case now.Hour() == c.SellHour && pos.Units > 0:
		if p := medianAt(history, c.SellHour, lookback, func(d prices.TimeseriesDatapoint) (int, bool) {
			return d.AvgHighPrice, d.HighPriceVolume > 0
		}); p > 0 {
			return []Order{{Side: Sell, Price: p, Quantity: pos.Units}}
		}
	}
	return nil
}

// medianAt is the median price at hour over the last days days of history.
func medianAt(history []prices.TimeseriesDatapoint, hour, days int, price func(prices.TimeseriesDatapoint) (int, bool)) int {
	var vals []int
	for j := len(history) - 1; j >= 0 && len(vals) < days; j-- {
		d := history[j]
		if d.Timestamp.UTC().Hour() == hour {
			if p, ok := price(d); ok {
				vals = append(vals, p)
			}
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Ints(vals)
	return vals[len(vals)/2]
}
