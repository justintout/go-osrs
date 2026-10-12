package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/skills"
)

type playerIn struct {
	Player string `json:"player" arg:"" jsonschema:"player name"`
	Mode   string `json:"mode,omitempty" jsonschema:"hiscores mode: regular (default), ironman, hardcore, ultimate, deadman, seasonal, tournament, fresh_start, 1_defence, or skiller"`
}

type playerOut struct {
	hiscores.Player
	CombatLevel int `json:"combat_level"`
}

func (o playerOut) text() string {
	var b strings.Builder
	overall := o.Overall()
	fmt.Fprintf(&b, "%s  combat %d  total %d  xp %s  rank %s\n\n", o.Name, o.CombatLevel, overall.Level, num(overall.XP), rank(overall.Rank))
	t := newTable(&b, "SKILL", "LEVEL", "XP", "RANK")
	for _, s := range o.Skills[1:] {
		t.row(s.Name, strconv.Itoa(s.Level), num(max(s.XP, 0)), rank(s.Rank))
	}
	t.flush()
	var ranked []hiscores.Activity
	for _, a := range o.Activities {
		if a.Score > 0 {
			ranked = append(ranked, a)
		}
	}
	if len(ranked) > 0 {
		b.WriteString("\n")
		t := newTable(&b, "ACTIVITY", "SCORE", "RANK")
		for _, a := range ranked {
			t.row(a.Name, num(int64(a.Score)), rank(a.Rank))
		}
		t.flush()
	}
	return b.String()
}

var hiscoresCmd = newCommand("hiscores", "look up a player's skills, bosses, and activities on the hiscores",
	func(ctx context.Context, d *deps, in playerIn) (playerOut, error) {
		if err := need("player", in.Player); err != nil {
			return playerOut{}, err
		}
		mode := hiscores.Mode(in.Mode)
		if mode == "" {
			mode = hiscores.Regular
		}
		p, err := d.hiscores.Lookup(ctx, in.Player, mode)
		if errors.Is(err, osrs.ErrNotFound) {
			return playerOut{}, fmt.Errorf("%s is not on the %s hiscores", in.Player, mode)
		}
		if err != nil {
			return playerOut{}, err
		}
		return playerOut{Player: *p, CombatLevel: p.CombatLevel()}, nil
	})

type accountIn struct {
	Player string `json:"player" arg:"" jsonschema:"player name"`
}

type accountOut struct {
	Name        string          `json:"name"`
	Mode        hiscores.Mode   `json:"mode"`
	Former      []hiscores.Mode `json:"former,omitempty"`
	CombatLevel int             `json:"combat_level"`
	TotalLevel  int             `json:"total_level"`
	TotalXP     int64           `json:"total_xp"`
}

func (o accountOut) text() string {
	s := fmt.Sprintf("%s: %s", o.Name, o.Mode)
	if len(o.Former) > 0 {
		var f []string
		for _, m := range o.Former {
			f = append(f, string(m))
		}
		s += " (formerly " + strings.Join(f, ", ") + ")"
	}
	return fmt.Sprintf("%s\ncombat %d  total %d  xp %s\n", s, o.CombatLevel, o.TotalLevel, num(o.TotalXP))
}

var accountCmd = newCommand("account", "infer a player's account type: regular, ironman, hardcore, or ultimate, and any mode they left",
	func(ctx context.Context, d *deps, in accountIn) (accountOut, error) {
		if err := need("player", in.Player); err != nil {
			return accountOut{}, err
		}
		a, err := d.hiscores.LookupAccount(ctx, in.Player)
		if errors.Is(err, osrs.ErrNotFound) {
			return accountOut{}, fmt.Errorf("%s is not on the hiscores", in.Player)
		}
		if err != nil {
			return accountOut{}, err
		}
		overall := a.Player.Overall()
		return accountOut{Name: a.Player.Name, Mode: a.Mode, Former: a.Former, CombatLevel: a.Player.CombatLevel(), TotalLevel: overall.Level, TotalXP: overall.XP}, nil
	})

type leaderboardIn struct {
	Skill    string `json:"skill,omitempty" jsonschema:"skill name, such as Overall or Woodcutting"`
	Activity string `json:"activity,omitempty" jsonschema:"boss or activity name as the hiscores write it, such as Vorkath or Clue Scrolls (all)"`
	Mode     string `json:"mode,omitempty" jsonschema:"hiscores mode, regular by default"`
	Page     int    `json:"page,omitempty" jsonschema:"page of 25 rows, starting at 1"`
}

type leaderboardOut struct {
	Board           string                   `json:"board"`
	Page            int                      `json:"page"`
	SkillEntries    []hiscores.SkillEntry    `json:"skill_entries,omitempty"`
	ActivityEntries []hiscores.ActivityEntry `json:"activity_entries,omitempty"`
}

func (o leaderboardOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s, page %d\n\n", o.Board, o.Page)
	if o.SkillEntries != nil {
		t := newTable(&b, "RANK", "NAME", "LEVEL", "XP")
		for _, e := range o.SkillEntries {
			t.row(num(int64(e.Rank)), e.Name, strconv.Itoa(e.Level), num(e.XP))
		}
		t.flush()
	} else {
		t := newTable(&b, "RANK", "NAME", "SCORE")
		for _, e := range o.ActivityEntries {
			t.row(num(int64(e.Rank)), e.Name, num(int64(e.Score)))
		}
		t.flush()
	}
	return b.String()
}

var leaderboardCmd = newCommand("leaderboard", "list one page of a skill or activity hiscores leaderboard",
	func(ctx context.Context, d *deps, in leaderboardIn) (leaderboardOut, error) {
		if (in.Skill == "") == (in.Activity == "") {
			return leaderboardOut{}, fmt.Errorf("set exactly one of skill or activity")
		}
		mode := hiscores.Mode(in.Mode)
		if mode == "" {
			mode = hiscores.Regular
		}
		page := max(in.Page, 1)
		if in.Skill != "" {
			id, name, err := skillID(in.Skill)
			if err != nil {
				return leaderboardOut{}, err
			}
			rows, err := d.hiscores.SkillLeaderboard(ctx, mode, id, page)
			return leaderboardOut{Board: name, Page: page, SkillEntries: rows}, err
		}
		id, name, err := activityID(ctx, d, mode, in.Activity)
		if err != nil {
			return leaderboardOut{}, err
		}
		rows, err := d.hiscores.ActivityLeaderboard(ctx, mode, id, page)
		return leaderboardOut{Board: name, Page: page, ActivityEntries: rows}, err
	})

// skillID maps a skill name to its hiscores table ID: Overall is 0, then
// skills in hiscores order.
func skillID(name string) (int, string, error) {
	if strings.EqualFold(name, "Overall") {
		return 0, "Overall", nil
	}
	for i, s := range skills.All {
		if strings.EqualFold(string(s), name) {
			return i + 1, string(s), nil
		}
	}
	return 0, "", fmt.Errorf("unknown skill %q", name)
}

// activityID finds an activity's table ID. Jagex adds activities over
// time, so the list comes from a live lookup of the top overall player.
func activityID(ctx context.Context, d *deps, mode hiscores.Mode, name string) (int, string, error) {
	top, err := d.hiscores.SkillLeaderboard(ctx, mode, 0, 1)
	if err != nil {
		return 0, "", err
	}
	if len(top) == 0 {
		return 0, "", fmt.Errorf("no players on the %s hiscores", mode)
	}
	p, err := d.hiscores.Lookup(ctx, top[0].Name, mode)
	if err != nil {
		return 0, "", err
	}
	if a, ok := p.Activity(name); ok {
		return a.ID, a.Name, nil
	}
	return 0, "", fmt.Errorf("unknown activity %q", name)
}
