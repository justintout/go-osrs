package market

import (
	"testing"

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
