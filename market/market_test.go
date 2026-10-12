package market

import (
	"math"
	"testing"
	"time"

	"github.com/justintout/go-osrs/prices"
)

func hour(price, volume int) prices.Average {
	return prices.Average{AvgHighPrice: price, HighPriceVolume: volume, AvgLowPrice: price, LowPriceVolume: volume}
}

func TestMovers(t *testing.T) {
	cat := prices.NewCatalog([]prices.Item{{ID: 1, Name: "Steady"}, {ID: 2, Name: "Crash"}, {ID: 3, Name: "Thin"}})
	var hours []prices.Averages
	for i := range 24 {
		hours = append(hours, prices.Averages{1: hour(1000+i%3, 50), 2: hour(1000+i%3, 50), 3: hour(1000, 1)})
	}
	hours = append(hours, prices.Averages{1: hour(1002, 50), 2: hour(700, 400), 3: hour(100, 1)})

	got := Movers(hours, cat, 10, 100)
	if len(got) != 2 || got[0].Name != "Crash" || got[0].Change > -0.29 || got[0].VolumeRatio < 3.9 {
		t.Errorf("movers = %+v", got)
	}
	if got[1].Name != "Steady" || got[1].Score > 2 {
		t.Errorf("steady = %+v", got[1])
	}
}

func TestCompare(t *testing.T) {
	cat := prices.NewCatalog([]prices.Item{{ID: 1, Name: "Item"}})
	before := []prices.Averages{{1: hour(100, 10)}, {1: hour(110, 10)}}
	after := []prices.Averages{{1: hour(150, 20)}}
	got := Compare(before, after, cat, 5, 100)
	if len(got) != 1 || got[0].Before != 105 || got[0].After != 150 {
		t.Errorf("compare = %+v", got)
	}
}

func TestBursts(t *testing.T) {
	var pts []prices.TimeseriesDatapoint
	for i := range 40 {
		pts = append(pts, prices.TimeseriesDatapoint{Timestamp: time.Unix(int64(i)*300, 0), AvgHighPrice: 260, HighPriceVolume: 20, AvgLowPrice: 250, LowPriceVolume: 30})
	}
	// One account buys most of its 13,000 limit at four times the price.
	pts[30].AvgHighPrice, pts[30].HighPriceVolume = 990, 9996
	// A big sell at the normal price is volume, not a burst.
	pts[31].LowPriceVolume = 9000

	got := Bursts(pts, 13000)
	if len(got) != 1 || got[0].Side != "buy" || got[0].Price != 990 || got[0].BuyLimitShare < 0.76 || got[0].BuyLimitShare > 0.77 {
		t.Errorf("bursts = %+v", got)
	}
}

func TestDailyCycles(t *testing.T) {
	cat := prices.NewCatalog([]prices.Item{{ID: 1, Name: "Cyclic"}, {ID: 2, Name: "Flat"}})
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	series := map[int][]prices.TimeseriesDatapoint{}
	for h := range 24 * 12 {
		ts := start.Add(time.Duration(h) * time.Hour)
		// Cheap at 06:00 UTC, dear at 18:00, with sells bunched at 05-07.
		price := 1000 + int(60*math.Sin(2*math.Pi*float64(ts.Hour()-12)/24))
		sell := 10
		if ts.Hour() >= 5 && ts.Hour() <= 7 {
			sell = 200
		}
		series[1] = append(series[1], prices.TimeseriesDatapoint{Timestamp: ts, AvgHighPrice: price, HighPriceVolume: 10, AvgLowPrice: price, LowPriceVolume: sell})
		series[2] = append(series[2], prices.TimeseriesDatapoint{Timestamp: ts, AvgHighPrice: 500 + h%2, HighPriceVolume: 10, AvgLowPrice: 500, LowPriceVolume: 10})
	}
	got := DailyCycles(series, cat, 6)
	if len(got) != 2 || got[0].Name != "Cyclic" {
		t.Fatalf("cycles = %+v", got)
	}
	c := got[0]
	if c.Low != 6 || c.High != 18 || c.Consistency != 1 || c.Edge < 0.09 || c.Dump != 5 || c.DumpShare < 0.6 {
		t.Errorf("cyclic = %+v", c)
	}
	if got[1].Edge > 0 {
		t.Errorf("flat item has edge %v", got[1].Edge)
	}
}

func TestWeeklyCycles(t *testing.T) {
	cat := prices.NewCatalog([]prices.Item{{ID: 1, Name: "Weekend"}})
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // a Sunday
	var pts []prices.TimeseriesDatapoint
	for d := range 7 * 15 {
		ts := start.AddDate(0, 0, d)
		price := 1000
		switch ts.Weekday() {
		case time.Wednesday:
			price = 900
		case time.Saturday:
			price = 1100
		}
		pts = append(pts, prices.TimeseriesDatapoint{Timestamp: ts, AvgHighPrice: price, HighPriceVolume: 10, AvgLowPrice: price, LowPriceVolume: 10})
	}
	got := WeeklyCycles(map[int][]prices.TimeseriesDatapoint{1: pts}, cat, 6)
	if len(got) != 1 || got[0].Low != int(time.Wednesday) || got[0].High != int(time.Saturday) || got[0].Wins != got[0].TestGroups {
		t.Errorf("weekly = %+v", got)
	}
}

func TestSeasonal(t *testing.T) {
	var pts []prices.TimeseriesDatapoint
	for d := range 365 * 4 {
		ts := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
		price := 1000
		switch ts.Month() {
		case time.July:
			price = 800 // a summer slump
		case time.December:
			price = 1200
		}
		pts = append(pts, prices.TimeseriesDatapoint{Timestamp: ts, AvgHighPrice: price, HighPriceVolume: 10, AvgLowPrice: price, LowPriceVolume: 10})
	}
	s, ok := Seasonal(pts)
	if !ok || s.Low != time.July || s.High != time.December || s.Wins != s.Tests || s.Years != 4 {
		t.Errorf("season = %+v", s)
	}
}
