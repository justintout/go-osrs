package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

type compareIn struct {
	Players string `json:"players" arg:"" jsonschema:"comma-separated player names"`
	Mode    string `json:"mode,omitempty" jsonschema:"hiscores mode, regular by default"`
}

type compareOut struct {
	Players []string `json:"players"`
	// Skills maps each skill to the players' levels, in Players order.
	Skills []compareRow `json:"skills"`
	// Errors maps players whose lookup failed to the reason.
	Errors map[string]string `json:"errors,omitempty"`
}

type compareRow struct {
	Skill  string  `json:"skill"`
	Levels []int   `json:"levels"`
	XP     []int64 `json:"xp"`
}

func (o compareOut) text() string {
	var b strings.Builder
	t := newTable(&b, append([]string{"SKILL"}, o.Players...)...)
	for _, r := range o.Skills {
		row := []string{r.Skill}
		best := slices.Max(r.XP)
		for i, l := range r.Levels {
			cell := strconv.Itoa(l)
			if r.XP[i] == best && best > 0 {
				cell += " *"
			}
			row = append(row, cell)
		}
		t.row(row...)
	}
	t.flush()
	for p, e := range o.Errors {
		fmt.Fprintf(&b, "%s: %s\n", p, e)
	}
	b.WriteString("* most XP\n")
	return b.String()
}

var compareCmd = newCommand("compare", "compare several players' levels side by side",
	func(ctx context.Context, d *deps, in compareIn) (compareOut, error) {
		var names []string
		for _, n := range strings.Split(in.Players, ",") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
		if len(names) < 2 {
			return compareOut{}, fmt.Errorf("give at least two comma-separated players")
		}
		mode := hiscores.Mode(in.Mode)
		if mode == "" {
			mode = hiscores.Regular
		}
		out := compareOut{}
		var found []*hiscores.Player
		for _, r := range d.hiscores.LookupMany(ctx, names, mode, 4) {
			switch {
			case errors.Is(r.Err, osrs.ErrNotFound):
				out.addError(r.Name, fmt.Sprintf("not on the %s hiscores", mode))
			case r.Err != nil:
				out.addError(r.Name, r.Err.Error())
			default:
				out.Players = append(out.Players, r.Player.Name)
				found = append(found, r.Player)
			}
		}
		if len(found) == 0 {
			return out, fmt.Errorf("no players found")
		}
		for i, s := range found[0].Skills {
			row := compareRow{Skill: s.Name}
			for _, p := range found {
				row.Levels = append(row.Levels, p.Skills[i].Level)
				row.XP = append(row.XP, p.Skills[i].XP)
			}
			out.Skills = append(out.Skills, row)
		}
		return out, nil
	})

func (o *compareOut) addError(player, msg string) {
	if o.Errors == nil {
		o.Errors = map[string]string{}
	}
	o.Errors[player] = msg
}
