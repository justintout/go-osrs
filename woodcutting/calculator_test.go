package woodcutting

import (
	"testing"
)

func TestCalculate(t *testing.T) {
	// Test case 1: Level 1 to 60
	actions := Calculate(XPForLevel(1), XPForLevel(60), []string{}, make(map[string]int))
	if len(actions) == 0 {
		t.Errorf("Expected actions, but got none")
	}

	// Test case 2: Disabled oaks
	actions = Calculate(XPForLevel(1), XPForLevel(60), []string{"Oak"}, make(map[string]int))
	for _, action := range actions {
		if action.Tree.Name == "Oak" {
			t.Errorf("Oak trees should be disabled, but were included in the calculation")
		}
	}

	// Test case 3: Start at level 30
	actions = Calculate(XPForLevel(30), XPForLevel(60), []string{}, make(map[string]int))
	if len(actions) > 0 && actions[0].Tree.Level < 30 {
		t.Errorf("Calculation started with a tree below the starting level")
	}

	// Test case 4: Override Oak level
	overrideLevels := map[string]int{"Oak": 20}
	actions = Calculate(XPForLevel(1), XPForLevel(60), []string{}, overrideLevels)
	oakActionFound := false
	for _, action := range actions {
		if action.Tree.Name == "Oak" {
			oakActionFound = true
			if LevelForXP(XPForLevel(1) + action.Count*25) < 20 {
				t.Errorf("Oak trees were cut before the overridden level of 20")
			}
		}
	}
	if !oakActionFound {
		t.Errorf("Oak action not found in the calculation with override")
	}
}