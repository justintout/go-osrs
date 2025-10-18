package hiscores

import (
	"testing"
)

func TestGetPlayer(t *testing.T) {
	player, err := GetPlayer("Zezima")
	if err != nil {
		t.Fatalf("Failed to get player data: %v", err)
	}

	if _, ok := player.Skills["Woodcutting"]; !ok {
		t.Error("Woodcutting skill not found in player data")
	}
}