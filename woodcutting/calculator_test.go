package woodcutting

import (
	"testing"
)

func TestCalculate(t *testing.T) {
	// Test case 1: Level 1 to 60
	actions := Calculate(XPForLevel(1), XPForLevel(60), []string{})
	if len(actions) == 0 {
		t.Errorf("Expected actions, but got none")
	}

	// Test case 2: Disabled oaks
	actions = Calculate(XPForLevel(1), XPForLevel(60), []string{"Oak"})
	for _, action := range actions {
		if action.Tree.Name == "Oak" {
			t.Errorf("Oak trees should be disabled, but were included in the calculation")
		}
	}

	// Test case 3: Start at level 30
	actions = Calculate(XPForLevel(30), XPForLevel(60), []string{})
	if len(actions) > 0 && actions[0].Tree.Level < 30 {
		t.Errorf("Calculation started with a tree below the starting level")
	}
}