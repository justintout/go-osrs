package bossing

import (
	"math"
	"testing"

	"github.com/justintout/go-osrs/wiki"
)

func TestEvaluate(t *testing.T) {
	b := Boss{Name: "Vorkath", Page: "Vorkath", KillsPerHour: 30, Drops: []wiki.Drop{
		{Item: "Bones", Source: "Vorkath#Quest", Probability: 1, QuantityLow: 1, QuantityHigh: 1, Rolls: 1},
		{Item: "Visage", Source: "Vorkath#Quest", Probability: 1, QuantityLow: 1, QuantityHigh: 1, Rolls: 1},
		{Item: "Superior dragon bones", Source: "Vorkath#Post-quest", Probability: 1, QuantityLow: 2, QuantityHigh: 2, Rolls: 1},
		{Item: "Coins", Source: "Vorkath#Post-quest", Probability: 0.5, QuantityLow: 100, QuantityHigh: 300, Rolls: 2},
		{Item: "Vorki", Source: "Vorkath#Post-quest", Probability: 1.0 / 3000, QuantityLow: 1, QuantityHigh: 1, Rolls: 1},
	}}
	value := map[string]int{"Bones": 1, "Visage": 5_000_000, "Superior dragon bones": 10_000, "Coins": 1}
	log := []wiki.LogItem{
		{Item: "Vorki", Sources: []wiki.LogSource{{Page: "Vorkath", Version: "Contract", Probability: 1}, {Page: "Vorkath", Version: "Post-quest", Probability: 1.0 / 3000}}},
		{Item: "Draconic visage", Sources: []wiki.LogSource{{Page: "Vorkath", Probability: 1.0 / 5000}}},
		{Item: "Jar of decay", Sources: []wiki.LogSource{{Page: "Vorkath", Rarity: "Rare"}}},
		{Item: "Zulrah scale", Sources: []wiki.LogSource{{Page: "Zulrah", Probability: 1}}},
	}
	r := Evaluate(b, func(i string) int { return value[i] }, map[string]bool{"Vorki": true}, log)

	// 2 bones at 10k, plus 0.5 * 2 rolls * 200 coins.
	if r.Source != "Vorkath#Post-quest" || r.GPPerKill != 20_200 || r.GPPerHour != 606_000 {
		t.Errorf("value = %+v", r)
	}
	if r.Pet != "Vorki" || math.Abs(r.HoursToPet-100) > 1e-6 {
		t.Errorf("pet = %+v", r)
	}
	if r.LogItems != 2 || r.LogItemsUnknown != 1 || r.HoursToLog < 200 || r.HoursToLog > 300 {
		t.Errorf("log = %+v", r)
	}
}
