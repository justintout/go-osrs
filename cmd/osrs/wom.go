package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/wom"
)

type womPlayerIn struct {
	Player string `json:"player" arg:"" jsonschema:"player name"`
}

type womPlayerOut struct {
	wom.Player
}

func (o womPlayerOut) text() string {
	return fmt.Sprintf("%s  %s %s  combat %d  xp %s\nehp %.1f  ehb %.1f  hours to max %.1f  hours to 200m all %.1f\nlast updated %s\n",
		o.DisplayName, o.Type, o.Build, o.CombatLevel, num(o.Exp), o.EHP, o.EHB, o.TTM, o.TT200m, o.UpdatedAt.Format(time.RFC3339))
}

var womPlayerCmd = newCommand("wom-player", "get a player's Wise Old Man profile: account type, build, EHP, EHB, and hours to max",
	func(ctx context.Context, d *deps, in womPlayerIn) (womPlayerOut, error) {
		if err := need("player", in.Player); err != nil {
			return womPlayerOut{}, err
		}
		p, err := d.wom.Player(ctx, in.Player)
		if errors.Is(err, osrs.ErrNotFound) {
			return womPlayerOut{}, untracked(in.Player)
		}
		if err != nil {
			return womPlayerOut{}, err
		}
		p.LatestSnapshot = nil
		return womPlayerOut{*p}, nil
	})

var womUpdateCmd = newCommand("wom-update", "ask Wise Old Man to refresh a player from the hiscores, starting tracking if needed",
	func(ctx context.Context, d *deps, in womPlayerIn) (womPlayerOut, error) {
		if err := need("player", in.Player); err != nil {
			return womPlayerOut{}, err
		}
		p, err := d.wom.UpdatePlayer(ctx, in.Player)
		if err != nil {
			return womPlayerOut{}, err
		}
		p.LatestSnapshot = nil
		return womPlayerOut{*p}, nil
	})

type womGainsIn struct {
	Player string `json:"player" arg:"" jsonschema:"player name"`
	Period string `json:"period,omitempty" jsonschema:"five_min, day, week (default), month, or year"`
}

type gainLine struct {
	Metric string  `json:"metric"`
	Kind   string  `json:"kind"`
	Gained float64 `json:"gained"`
	End    float64 `json:"end"`
}

type womGainsOut struct {
	Player   string     `json:"player"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
	Gains    []gainLine `json:"gains"`
}

func (o womGainsOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s gains %s to %s\n\n", o.Player, o.StartsAt.Format(time.DateOnly), o.EndsAt.Format(time.DateOnly))
	if len(o.Gains) == 0 {
		b.WriteString("no progress in this period\n")
		return b.String()
	}
	t := newTable(&b, "METRIC", "KIND", "GAINED", "NOW")
	for _, g := range o.Gains {
		t.row(g.Metric, g.Kind, num(int64(g.Gained)), num(int64(g.End)))
	}
	t.flush()
	return b.String()
}

var womGainsCmd = newCommand("wom-gains", "list what a player gained over a period: XP per skill, boss kills, and activity scores",
	func(ctx context.Context, d *deps, in womGainsIn) (womGainsOut, error) {
		if err := need("player", in.Player); err != nil {
			return womGainsOut{}, err
		}
		period := wom.Period(in.Period)
		if period == "" {
			period = wom.Week
		}
		g, err := d.wom.Gains(ctx, in.Player, period)
		if errors.Is(err, osrs.ErrNotFound) {
			return womGainsOut{}, untracked(in.Player)
		}
		if err != nil {
			return womGainsOut{}, err
		}
		out := womGainsOut{Player: in.Player, StartsAt: g.StartsAt, EndsAt: g.EndsAt}
		for m, s := range g.Data.Skills {
			if s.Experience.Gained > 0 {
				out.Gains = append(out.Gains, gainLine{m, "skill xp", float64(s.Experience.Gained), float64(s.Experience.End)})
			}
		}
		for m, s := range g.Data.Bosses {
			if s.Kills.Gained > 0 {
				out.Gains = append(out.Gains, gainLine{m, "boss kills", float64(s.Kills.Gained), float64(s.Kills.End)})
			}
		}
		for m, s := range g.Data.Activities {
			if s.Score.Gained > 0 {
				out.Gains = append(out.Gains, gainLine{m, "activity score", float64(s.Score.Gained), float64(s.Score.End)})
			}
		}
		sort.Slice(out.Gains, func(i, j int) bool {
			if out.Gains[i].Kind != out.Gains[j].Kind {
				return out.Gains[i].Kind > out.Gains[j].Kind
			}
			return out.Gains[i].Gained > out.Gains[j].Gained
		})
		return out, nil
	})

type idIn struct {
	ID int `json:"id" arg:"" jsonschema:"Wise Old Man ID"`
}

type womGroupOut struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	ClanChat    string   `json:"clan_chat"`
	Description string   `json:"description"`
	MemberCount int      `json:"member_count"`
	Members     []string `json:"members"`
}

func (o womGroupOut) text() string {
	return fmt.Sprintf("%s (id %d)  clan chat %q  %d members\n%s\n\n%s\n", o.Name, o.ID, o.ClanChat, o.MemberCount, o.Description, strings.Join(o.Members, ", "))
}

var womGroupCmd = newCommand("wom-group", "get a Wise Old Man group and its members",
	func(ctx context.Context, d *deps, in idIn) (womGroupOut, error) {
		g, err := d.wom.Group(ctx, in.ID)
		if err != nil {
			return womGroupOut{}, err
		}
		out := womGroupOut{ID: g.ID, Name: g.Name, ClanChat: g.ClanChat, Description: g.Description, MemberCount: g.MemberCount}
		for _, m := range g.Memberships {
			out.Members = append(out.Members, m.Player.DisplayName)
		}
		return out, nil
	})

type standing struct {
	Player string `json:"player"`
	Team   string `json:"team,omitempty"`
	Gained int64  `json:"gained"`
}

type womCompetitionOut struct {
	ID       int        `json:"id"`
	Title    string     `json:"title"`
	Metric   string     `json:"metric"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
	Standing []standing `json:"standings"`
}

func (o womCompetitionOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)  %s to %s\n\n", o.Title, o.Metric, o.StartsAt.Format(time.DateOnly), o.EndsAt.Format(time.DateOnly))
	t := newTable(&b, "#", "PLAYER", "TEAM", "GAINED")
	for i, s := range o.Standing {
		t.row(fmt.Sprint(i+1), s.Player, s.Team, num(s.Gained))
	}
	t.flush()
	return b.String()
}

var womCompetitionCmd = newCommand("wom-competition", "get a Wise Old Man competition's standings",
	func(ctx context.Context, d *deps, in idIn) (womCompetitionOut, error) {
		c, err := d.wom.Competition(ctx, in.ID)
		if err != nil {
			return womCompetitionOut{}, err
		}
		out := womCompetitionOut{ID: c.ID, Title: c.Title, Metric: c.Metric, StartsAt: c.StartsAt, EndsAt: c.EndsAt}
		for _, p := range c.Participations {
			out.Standing = append(out.Standing, standing{p.Player.DisplayName, p.TeamName, p.Progress.Gained})
		}
		return out, nil
	})

func untracked(player string) error {
	return fmt.Errorf("%s is not tracked by Wise Old Man; run wom-update to start tracking", player)
}
