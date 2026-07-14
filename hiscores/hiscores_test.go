package hiscores

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// cannedHiscores is a minimal but well-formed index_lite response: one
// "rank,level,xp" line per skill in skillNames order. Woodcutting is the
// 10th skill (index 9).
func cannedHiscores() string {
	lines := make([]string, len(skillNames))
	for i := range skillNames {
		rank := 1000 + i
		level := 50 + i
		xp := 100000 + i*1000
		lines[i] = fmt.Sprintf("%d,%d,%d", rank, level, xp)
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestGetPlayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, cannedHiscores())
	}))
	defer srv.Close()

	old := hiscoresURL
	hiscoresURL = srv.URL + "?player=%s"
	defer func() { hiscoresURL = old }()

	player, err := GetPlayer("Zezima")
	if err != nil {
		t.Fatalf("Failed to get player data: %v", err)
	}

	wc, ok := player.Skills["Woodcutting"]
	if !ok {
		t.Fatal("Woodcutting skill not found in player data")
	}
	// Woodcutting is index 9 in skillNames, so level == 50+9, xp == 100000+9000.
	if wc.Level != 59 {
		t.Errorf("Woodcutting level = %d, want 59", wc.Level)
	}
	if wc.XP != 109000 {
		t.Errorf("Woodcutting xp = %d, want 109000", wc.XP)
	}
}

func TestGetPlayerNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	old := hiscoresURL
	hiscoresURL = srv.URL + "?player=%s"
	defer func() { hiscoresURL = old }()

	if _, err := GetPlayer("Nonexistent Player 123"); err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

// TestGetPlayerLive hits the real OSRS hiscores API. It is skipped by default
// and only runs when OSRS_LIVE_HISCORES is set, so the normal test run stays
// hermetic and offline-friendly.
func TestGetPlayerLive(t *testing.T) {
	if os.Getenv("OSRS_LIVE_HISCORES") == "" {
		t.Skip("set OSRS_LIVE_HISCORES to run the live hiscores test")
	}
	player, err := GetPlayer("Zezima")
	if err != nil {
		t.Fatalf("Failed to get player data: %v", err)
	}
	if _, ok := player.Skills["Woodcutting"]; !ok {
		t.Error("Woodcutting skill not found in player data")
	}
}
