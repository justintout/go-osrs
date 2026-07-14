package woodcutting

import (
	"testing"
	"time"
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

// TestCalculateTerminatesToLevel99 is a regression test for an infinite loop
// that occurred whenever the target was level 99. Once the current level
// reached 75 (Magic, the highest-level tree), no higher tree existed, the
// next-breakpoint search fell through to XPForLevel(100) == 0, and the loop
// reset currentXp to 0 and spun forever. Each case must complete on its own;
// we do not rely on the test binary's overall timeout.
func TestCalculateTerminatesToLevel99(t *testing.T) {
	starts := []int{1, 75, 76, 90}
	for _, start := range starts {
		start := start
		t.Run("start_"+itoa(start), func(t *testing.T) {
			done := make(chan []Action, 1)
			go func() {
				done <- Calculate(XPForLevel(start), XPForLevel(99), []string{}, make(map[string]int))
			}()
			select {
			case actions := <-done:
				if len(actions) == 0 {
					t.Fatalf("start %d -> 99: expected actions, got none", start)
				}
				last := actions[len(actions)-1]
				if last.Tree.Name != "Magic" {
					t.Errorf("start %d -> 99: expected final tree to be Magic, got %q", start, last.Tree.Name)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("start %d -> 99: Calculate did not terminate within 2s (infinite loop regression)", start)
			}
		})
	}
}

// itoa is a tiny helper to avoid importing strconv just for subtest names.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}