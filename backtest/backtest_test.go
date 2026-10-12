package backtest

import (
	"testing"
	"time"

	"github.com/justintout/go-osrs/prices"
)

// series builds hourly data that is cheap at 06:00 and dear at 18:00 UTC.
func series(days int, swing int) []prices.TimeseriesDatapoint {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var s []prices.TimeseriesDatapoint
	for h := range days * 24 {
		ts := start.Add(time.Duration(h) * time.Hour)
		p := 1000
		switch ts.Hour() {
		case 6:
			p -= swing
		case 18:
			p += swing
		}
		s = append(s, prices.TimeseriesDatapoint{Timestamp: ts, AvgLowPrice: p - 5, LowPriceVolume: 1000, AvgHighPrice: p + 5, HighPriceVolume: 1000})
	}
	return s
}

func TestCycleProfitsWhenSwingBeatsTax(t *testing.T) {
	r := Run(series(10, 100), Cycle{BuyHour: 6, SellHour: 18, Quantity: 50}, Config{ItemID: 1, BuyLimit: 500})
	// From day 2, buy 50 at 895 and sell at 1105 less 22 tax: 188 each.
	if r.Profit != 9*50*188 || r.WinningDays != 9 || r.Unrealized != 0 {
		t.Errorf("result = %+v", r)
	}
}

func TestCycleLosesWhenTaxEatsSwing(t *testing.T) {
	r := Run(series(10, 0), Cycle{BuyHour: 6, SellHour: 18, Quantity: 50}, Config{ItemID: 1, BuyLimit: 500})
	if r.Profit >= 0 {
		t.Errorf("profit = %d, want a loss after tax", r.Profit)
	}
}

func TestParticipationAndBuyLimit(t *testing.T) {
	s := series(3, 100)
	for i := range s {
		s[i].LowPriceVolume = 100 // 10% participation allows 10 per window
	}
	r := Run(s, Cycle{BuyHour: 6, SellHour: 18, Quantity: 1000}, Config{ItemID: 1, BuyLimit: 8})
	for _, tr := range r.Trades {
		if tr.Side == "buy" && tr.Quantity != 8 {
			t.Errorf("buy of %d, want the buy limit of 8", tr.Quantity)
		}
	}
}
