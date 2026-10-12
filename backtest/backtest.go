// Package backtest replays trading strategies against Grand Exchange
// price history with a conservative fill model, tax, and buy limits, to
// test whether a strategy would have made money.
package backtest

import (
	"time"

	"github.com/justintout/go-osrs/prices"
)

// Side is buy or sell.
type Side int

const (
	Buy Side = iota
	Sell
)

// Order is a limit order good for one window. A Market sell fills at the
// window's average instant-sell price, to exit a position.
type Order struct {
	Side     Side
	Price    int
	Quantity int
	Market   bool
}

// Position is what the strategy holds.
type Position struct {
	Units int
	// Cost is the coins paid for Units.
	Cost int
	// Since is when the oldest held unit was bought.
	Since time.Time
}

// Strategy places orders for window i. It sees only history[:i], so it
// cannot use the window it is trading in.
type Strategy interface {
	Orders(i int, history []prices.TimeseriesDatapoint, pos Position) []Order
}

// Config sets the market rules.
type Config struct {
	ItemID int
	// BuyLimit is the item's 4-hour Grand Exchange buy limit.
	BuyLimit int
	// Participation is the most of one window's volume on a side that
	// your orders can take, 0.1 by default. Other traders compete for
	// the rest.
	Participation float64
	// Capital caps the coins tied up in a position. 0 means no cap.
	Capital int
	// From is when trading starts. Earlier windows only feed the
	// strategy's history, so a strategy tuned on them is not scored on
	// them.
	From time.Time
}

// Trade is one fill.
type Trade struct {
	Time     time.Time `json:"time"`
	Side     string    `json:"side"`
	Price    int       `json:"price"`
	Quantity int       `json:"quantity"`
	// Profit is set on sells: proceeds after tax less the units' cost.
	Profit int `json:"profit,omitempty"`
}

// Result summarizes a run.
type Result struct {
	Trades []Trade `json:"trades"`
	// Profit is realized profit after tax. Units still held at the end
	// are valued at the last instant-sell price after tax.
	Profit     int `json:"profit"`
	Unrealized int `json:"unrealized,omitempty"`
	// MaxCapital is the most coins tied up at once.
	MaxCapital int `json:"max_capital"`
	// ROI is Profit over MaxCapital.
	ROI float64 `json:"roi"`
	// Days counts calendar days from From to the end and WinningDays
	// those with positive realized profit.
	Days        int `json:"days"`
	WinningDays int `json:"winning_days"`
}

// Run replays strategy over series, oldest first.
//
// A limit buy fills when the window's average instant-sell price is at or
// below the order price, and a limit sell when the average instant-buy
// price is at or above it, each up to Participation of that side's volume.
// Fills are at the order price. Sales pay the GE tax, and buys respect the
// 4-hour buy limit.
func Run(series []prices.TimeseriesDatapoint, strategy Strategy, cfg Config) Result {
	if cfg.Participation <= 0 {
		cfg.Participation = 0.1
	}
	var (
		r      Result
		pos    Position
		bought []Trade // buys within the last 4 hours, for the buy limit
		daily  = map[string]int{}
	)
	for i, w := range series {
		if w.Timestamp.Before(cfg.From) {
			continue
		}
		for _, o := range strategy.Orders(i, series[:i], pos) {
			switch {
			case o.Side == Buy && !o.Market:
				recent := 0
				kept := bought[:0]
				for _, b := range bought {
					if w.Timestamp.Sub(b.Time) < 4*time.Hour {
						kept = append(kept, b)
						recent += b.Quantity
					}
				}
				bought = kept
				qty := min(o.Quantity, int(cfg.Participation*float64(w.LowPriceVolume)))
				if cfg.BuyLimit > 0 {
					qty = min(qty, cfg.BuyLimit-recent)
				}
				if cfg.Capital > 0 && o.Price > 0 {
					qty = min(qty, (cfg.Capital-pos.Cost)/o.Price)
				}
				if qty <= 0 || w.LowPriceVolume == 0 || w.AvgLowPrice > o.Price {
					continue
				}
				if pos.Units == 0 {
					pos.Since = w.Timestamp
				}
				pos.Units += qty
				pos.Cost += qty * o.Price
				t := Trade{Time: w.Timestamp, Side: "buy", Price: o.Price, Quantity: qty}
				bought = append(bought, t)
				r.Trades = append(r.Trades, t)
				r.MaxCapital = max(r.MaxCapital, pos.Cost)
			case o.Side == Sell && pos.Units > 0:
				price, volume := o.Price, w.HighPriceVolume
				filled := w.HighPriceVolume > 0 && w.AvgHighPrice >= o.Price
				if o.Market {
					price, volume, filled = w.AvgLowPrice, w.LowPriceVolume, w.LowPriceVolume > 0
				}
				qty := min(o.Quantity, pos.Units, int(cfg.Participation*float64(volume)))
				if !filled || qty <= 0 {
					continue
				}
				cost := pos.Cost * qty / pos.Units
				profit := qty*prices.AfterTax(cfg.ItemID, price) - cost
				pos.Units -= qty
				pos.Cost -= cost
				r.Trades = append(r.Trades, Trade{Time: w.Timestamp, Side: "sell", Price: price, Quantity: qty, Profit: profit})
				r.Profit += profit
				daily[w.Timestamp.UTC().Format(time.DateOnly)] += profit
			}
		}
	}
	if pos.Units > 0 && len(series) > 0 {
		last := series[len(series)-1].AvgLowPrice
		r.Unrealized = pos.Units*prices.AfterTax(cfg.ItemID, last) - pos.Cost
	}
	if len(series) > 0 {
		start := series[0].Timestamp
		if cfg.From.After(start) {
			start = cfg.From
		}
		r.Days = int(series[len(series)-1].Timestamp.Sub(start).Hours()/24) + 1
	}
	for _, p := range daily {
		if p > 0 {
			r.WinningDays++
		}
	}
	if r.MaxCapital > 0 {
		r.ROI = float64(r.Profit) / float64(r.MaxCapital)
	}
	return r
}
