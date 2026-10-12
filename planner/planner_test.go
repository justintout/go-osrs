package planner

import (
	"math"
	"testing"

	"github.com/justintout/go-osrs/skills"
)

func names(p Plan) []string {
	var out []string
	for _, s := range p.Steps {
		out = append(out, s.Method.Name)
	}
	return out
}

func TestRebalancesAtUnlocks(t *testing.T) {
	methods := []Method{
		{Name: "slow", Level: 1, XP: 10, Rates: []Rate{{0, 10_000}}},
		{Name: "fast", Level: 50, XP: 20, Rates: []Rate{{0, 40_000}}},
	}
	p, err := Solve(methods, 0, skills.XPForLevel(70), Goal{Objective: Hours})
	if err != nil {
		t.Fatal(err)
	}
	if got := names(p); len(got) != 2 || got[0] != "slow" || got[1] != "fast" || p.Steps[1].FromXP != skills.XPForLevel(50) {
		t.Errorf("steps = %+v", p.Steps)
	}
}

func TestHoursAcrossRateBands(t *testing.T) {
	// One method that doubles its rate at 1,000 XP.
	m := Method{Name: "teaks", Level: 1, XP: 10, Rates: []Rate{{0, 1_000}, {1_000, 2_000}}}
	p, err := Solve([]Method{m}, 0, 3_000, Goal{Objective: Hours})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 1 || math.Abs(p.Hours-2) > 1e-9 || p.Actions != 300 {
		t.Errorf("plan = %+v", p)
	}
}

func TestCapsGoWhereTheySaveMost(t *testing.T) {
	split, end := 10_000, 20_000
	// "rare" beats "base" everywhere, a little before the split and by a
	// lot after it, but its cap covers only one half. A greedy pass spends
	// it first; the optimum saves it for after the split.
	methods := []Method{
		{Name: "base", Level: 1, XP: 1, Rates: []Rate{{0, 20_000}}},
		{Name: "rare", Level: 1, XP: 1, Rates: []Rate{{0, 30_000}, {split, 100_000}}, MaxActions: end - split},
	}
	p, err := Solve(methods, 0, end, Goal{Objective: Hours})
	if err != nil {
		t.Fatal(err)
	}
	if got := names(p); len(got) != 2 || got[0] != "base" || got[1] != "rare" || p.Steps[1].FromXP != split {
		t.Errorf("steps = %+v", p.Steps)
	}
}

func TestBalancedTradesTimeForCoins(t *testing.T) {
	methods := []Method{
		{Name: "cheap", Level: 1, XP: 10, GP: 0, Rates: []Rate{{0, 50_000}}},
		{Name: "fast", Level: 1, XP: 10, GP: -50, Rates: []Rate{{0, 100_000}}},
	}
	// fast costs 5 gp/XP and saves 1/100k hours per XP: worth it only if
	// an hour is worth more than 500k.
	for _, tc := range []struct {
		gpPerHour float64
		want      string
	}{{100_000, "cheap"}, {5_000_000, "fast"}} {
		p, err := Solve(methods, 0, 100_000, Goal{Objective: Balanced, GPPerHour: tc.gpPerHour})
		if err != nil {
			t.Fatal(err)
		}
		if got := names(p); len(got) != 1 || got[0] != tc.want {
			t.Errorf("gp/hr %v: %v, want %s", tc.gpPerHour, got, tc.want)
		}
	}
}

func TestErrors(t *testing.T) {
	if _, err := Solve([]Method{{Name: "late", Level: 50, XP: 1}}, 0, 1_000, Goal{}); err == nil {
		t.Error("no method at level 1: want error")
	}
	if _, err := Solve([]Method{{Name: "capped", Level: 1, XP: 1, MaxActions: 10}}, 0, 1_000, Goal{}); err == nil {
		t.Error("caps cannot cover: want error")
	}
}
