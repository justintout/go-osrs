package woodcutting

import (
	"testing"

	"github.com/justintout/go-osrs/skills"
)

func TestCalculate(t *testing.T) {
	actions, err := Calculate(0, skills.XPForLevel(99), []string{"Yew"}, map[string]int{"Willow": 40})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range actions {
		names = append(names, a.Tree.Name)
	}
	// Willow waits until 40, so Teak (35) takes over first; Yew is skipped.
	want := []string{"Normal", "Oak", "Teak", "Maple", "Mahogany", "Magic"}
	if len(names) != len(want) {
		t.Fatalf("trees = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("trees = %v, want %v", names, want)
		}
	}
}
