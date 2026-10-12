package planner

import (
	"testing"

	"github.com/justintout/go-osrs/skills"
)

func TestPlan(t *testing.T) {
	methods := []Method{
		{Name: "cheap", Level: 1, XP: 10, GP: -1},
		{Name: "fast", Level: 20, XP: 50, GP: -100},
		{Name: "profit", Level: 40, XP: 20, GP: 5},
	}
	from, to := skills.XPForLevel(1), skills.XPForLevel(60)

	steps, err := Plan(methods, from, to, FewestActions)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].Method.Name != "cheap" || steps[1].Method.Name != "fast" || steps[1].ToXP < to {
		t.Errorf("fewest actions = %+v", steps)
	}

	steps, err = Plan(methods, from, to, BestGP)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].Method.Name != "cheap" || steps[1].Method.Name != "profit" || steps[1].FromXP < skills.XPForLevel(40) {
		t.Errorf("best gp = %+v", steps)
	}

	capped := []Method{{Name: "rare", Level: 1, XP: 100, MaxActions: 10}, {Name: "slow", Level: 1, XP: 1}}
	steps, err = Plan(capped, 0, 2000, FewestActions)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].Actions != 10 || steps[1].Method.Name != "slow" || steps[1].Actions != 1000 {
		t.Errorf("capped = %+v", steps)
	}

	if _, err := Plan([]Method{{Name: "late", Level: 50, XP: 1}}, from, to, FewestActions); err == nil {
		t.Error("no method at level 1: want error")
	}
}
