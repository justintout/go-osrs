package hiscores_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/hiscores"
)

// boards maps a hiscores path segment to the Overall XP it reports for the
// test player. A missing segment means the player is not on that board.
func fakeHiscores(t *testing.T, boards map[string]int64) *hiscores.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("player"); got != "Lynx Titan" {
			t.Errorf("player = %q", got)
		}
		board := strings.TrimPrefix(strings.Split(r.URL.Path, "/")[1], "m=")
		xp, ok := boards[board]
		if !ok {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(hiscores.Player{
			Name: "Lynx Titan",
			Skills: []hiscores.Skill{
				{ID: 0, Name: "Overall", Rank: 1, Level: 2376, XP: xp},
				{ID: 9, Name: "Woodcutting", Rank: 5, Level: 99, XP: 200_000_000},
			},
			Activities: []hiscores.Activity{{ID: 10, Name: "Clue Scrolls (medium)", Rank: -1, Score: -1}},
		})
	}))
	t.Cleanup(srv.Close)
	return hiscores.NewClient("test", hiscores.WithBaseURL(srv.URL))
}

func TestLookup(t *testing.T) {
	c := fakeHiscores(t, map[string]int64{"hiscore_oldschool": 4_600_000_000})
	p, err := c.Lookup(context.Background(), "Lynx Titan", hiscores.Regular)
	if err != nil {
		t.Fatal(err)
	}
	if wc, ok := p.Skill("woodcutting"); !ok || wc.Level != 99 {
		t.Errorf("Skill(woodcutting) = %+v, %v", wc, ok)
	}
	if p.Overall().XP != 4_600_000_000 {
		t.Errorf("Overall XP = %d", p.Overall().XP)
	}
	if _, err := c.Lookup(context.Background(), "Lynx Titan", hiscores.Ironman); !errors.Is(err, osrs.ErrNotFound) {
		t.Errorf("ironman lookup err = %v, want ErrNotFound", err)
	}
}

func TestLookupAccount(t *testing.T) {
	const (
		reg  = "hiscore_oldschool"
		iron = "hiscore_oldschool_ironman"
		hc   = "hiscore_oldschool_hardcore_ironman"
		uim  = "hiscore_oldschool_ultimate"
	)
	tests := []struct {
		name   string
		boards map[string]int64
		mode   hiscores.Mode
		former []hiscores.Mode
	}{
		{"regular", map[string]int64{reg: 100}, hiscores.Regular, nil},
		{"ironman", map[string]int64{reg: 100, iron: 100}, hiscores.Ironman, nil},
		{"hardcore", map[string]int64{reg: 100, iron: 100, hc: 100}, hiscores.Hardcore, nil},
		{"dead hardcore", map[string]int64{reg: 100, iron: 100, hc: 60}, hiscores.Ironman, []hiscores.Mode{hiscores.Hardcore}},
		{"ultimate", map[string]int64{reg: 100, iron: 100, uim: 100}, hiscores.Ultimate, nil},
		{"de-ulted", map[string]int64{reg: 100, iron: 100, uim: 40}, hiscores.Ironman, []hiscores.Mode{hiscores.Ultimate}},
		{"de-ironed hardcore", map[string]int64{reg: 100, iron: 70, hc: 70}, hiscores.Regular, []hiscores.Mode{hiscores.Ironman, hiscores.Hardcore}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acct, err := fakeHiscores(t, tt.boards).LookupAccount(context.Background(), "Lynx Titan")
			if err != nil {
				t.Fatal(err)
			}
			if acct.Mode != tt.mode || !slices.Equal(acct.Former, tt.former) {
				t.Errorf("got mode %s former %v, want %s %v", acct.Mode, acct.Former, tt.mode, tt.former)
			}
		})
	}
}

func TestLeaderboards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := "testdata/overall.html"
		if r.URL.Query().Get("category_type") == "1" {
			file = "testdata/activity.html"
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
	}))
	defer srv.Close()
	c := hiscores.NewClient("test", hiscores.WithBaseURL(srv.URL))
	ctx := context.Background()

	skills, err := c.SkillLeaderboard(ctx, hiscores.Regular, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := hiscores.SkillEntry{Rank: 1, Name: "FasT 07", Level: 2376, XP: 4_800_000_000}
	if len(skills) != 2 || skills[0] != want {
		t.Errorf("skills = %+v, want first %+v", skills, want)
	}

	acts, err := c.ActivityLeaderboard(ctx, hiscores.Regular, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 2 || acts[1] != (hiscores.ActivityEntry{Rank: 2, Name: "UrSeedVault", Score: 96_825}) {
		t.Errorf("activities = %+v", acts)
	}

	if _, err := c.SkillLeaderboard(ctx, hiscores.Regular, 0, 0); err == nil {
		t.Error("page 0: want error")
	}
}
