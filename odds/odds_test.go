package odds

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestOdds(t *testing.T) {
	const p = 1.0 / 3000
	if got := ChanceWithin(p, 3000); !near(got, 0.6322, 1e-3) {
		t.Errorf("ChanceWithin = %v", got)
	}
	if got := DryChance(p, 9000); !near(got, 0.0498, 1e-3) {
		t.Errorf("DryChance = %v", got)
	}
	if got := KillsFor(p, 0.9); got != 6907 {
		t.Errorf("KillsFor = %v", got)
	}
	if got := PerKill(1.0/128, 2); !near(got, 0.01556, 1e-4) {
		t.Errorf("PerKill = %v", got)
	}
}

func TestExpectedKillsForAll(t *testing.T) {
	// One drop reduces to 1/p.
	if got := ExpectedKillsForAll(0.01); !near(got, 100, 0.01) {
		t.Errorf("single = %v", got)
	}
	// By inclusion-exclusion, E = 1/p1 + 1/p2 - 1/(1-(1-p1)(1-p2))
	// = 200 - 1/0.0199 = 149.75.
	if got := ExpectedKillsForAll(0.01, 0.01); !near(got, 149.75, 0.05) {
		t.Errorf("pair = %v", got)
	}
}
