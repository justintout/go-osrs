package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/justintout/go-osrs/planner"
	"github.com/justintout/go-osrs/profit"
	"github.com/justintout/go-osrs/skills"
	"github.com/justintout/go-osrs/woodcutting"
)

type trainIn struct {
	Skill      string `json:"skill" arg:"" jsonschema:"skill to train, such as Fletching"`
	Player     string `json:"player,omitempty" jsonschema:"start from this player's current XP"`
	From       int    `json:"from,omitempty" jsonschema:"start level when no player is given, 1 by default"`
	To         int    `json:"to,omitempty" jsonschema:"target level, 99 by default"`
	By         string `json:"by,omitempty" jsonschema:"gp (default) for the most coins per XP, which favors slow profitable methods, or xp for the fewest actions"`
	MinVolume  int    `json:"min_volume,omitempty" jsonschema:"minimum 24h trades for each recipe item, 500 by default"`
	MarketDays int    `json:"market_days,omitempty" jsonschema:"cap each recipe at this many days of its output's whole-market 24h volume, 1 by default"`
}

type trainOut struct {
	Skill   string         `json:"skill"`
	FromXP  int            `json:"from_xp"`
	ToXP    int            `json:"to_xp"`
	Goal    string         `json:"goal"`
	Steps   []planner.Step `json:"steps"`
	TotalGP int            `json:"total_gp"`
}

func (o trainOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s level %d to %d, by %s\n\n", o.Skill, skills.Level(o.FromXP), skills.Level(o.ToXP), o.Goal)
	t := newTable(&b, "LEVELS", "METHOD", "XP EACH", "ACTIONS", "GP EACH", "GP")
	for _, s := range o.Steps {
		levels := fmt.Sprintf("%d-%d", skills.Level(s.FromXP), skills.Level(s.ToXP))
		t.row(levels, s.Method.Name, fmt.Sprintf("%g", s.Method.XP), num(int64(s.Actions)), fmt.Sprintf("%.0f", s.Method.GP), num(int64(s.GP)))
	}
	t.flush()
	fmt.Fprintf(&b, "\ntotal gp %s\nrecipe steps are capped at the market's daily volume; prices move as you trade\n", signed(o.TotalGP))
	return b.String()
}

var trainCmd = newCommand("train", "plan training a skill to a target level: the cheapest or most profitable methods at live prices, or the fewest actions",
	func(ctx context.Context, d *deps, in trainIn) (trainOut, error) {
		if err := need("skill", in.Skill); err != nil {
			return trainOut{}, err
		}
		_, skill, err := skillID(in.Skill)
		if err != nil || skill == "Overall" {
			return trainOut{}, fmt.Errorf("unknown skill %q", in.Skill)
		}
		goal, goalName := planner.BestGP, "gp"
		switch in.By {
		case "", "gp":
		case "xp":
			goal, goalName = planner.FewestActions, "xp"
		default:
			return trainOut{}, fmt.Errorf("by must be gp or xp")
		}

		fromXP := skills.XPForLevel(max(in.From, 1))
		if in.Player != "" {
			levels, err := playerXP(ctx, d, in.Player)
			if err != nil {
				return trainOut{}, err
			}
			fromXP = levels[strings.ToLower(skill)]
		}
		toXP := skills.XPForLevel(orDefault(in.To, 99))
		if fromXP >= toXP {
			return trainOut{}, fmt.Errorf("already at or past level %d", orDefault(in.To, 99))
		}

		methods, err := trainingMethods(ctx, d, skill, goal, orDefault(in.MinVolume, defaultMinVolume), orDefault(in.MarketDays, 1))
		if err != nil {
			return trainOut{}, err
		}
		steps, err := planner.Plan(methods, fromXP, toXP, goal)
		if err != nil {
			return trainOut{}, err
		}
		out := trainOut{Skill: skill, FromXP: fromXP, ToXP: toXP, Goal: goalName, Steps: steps}
		for _, s := range steps {
			out.TotalGP += s.GP
		}
		return out, nil
	})

// trainingMethods lists ways to train skill. Woodcutting uses its tree
// table. Other skills use wiki recipes, priced at live prices, which
// covers production skills like Fletching, Smithing, and Herblore. Each
// recipe is capped at marketDays of its output's 24h volume, since a
// plan that floods the market cannot sell at the quoted price. Recipes
// that make the same item share that market, so only the best one for
// goal is kept.
func trainingMethods(ctx context.Context, d *deps, skill string, goal planner.Goal, minVolume, marketDays int) ([]planner.Method, error) {
	if skill == string(skills.Woodcutting) {
		return woodcutting.Methods(), nil
	}
	recipes, err := d.wiki.Recipes(ctx)
	if err != nil {
		return nil, err
	}
	cat, latest, vols, err := market(ctx, d)
	if err != nil {
		return nil, err
	}
	better := func(a, b planner.Method) bool {
		if goal == planner.BestGP {
			return a.GP/a.XP > b.GP/b.XP
		}
		return a.XP > b.XP
	}
	byOutput := map[string]planner.Method{}
	for _, rp := range profit.PriceRecipes(recipes, cat, latest, vols, minVolume) {
		for _, s := range rp.Recipe.Skills {
			if !strings.EqualFold(s.Name, skill) || s.XP <= 0 {
				continue
			}
			name := rp.Recipe.Output.Name
			m := planner.Method{Name: name, Level: s.Level, XP: s.XP, GP: float64(rp.Profit), MaxActions: rp.OutputVolume24h * marketDays}
			if cur, ok := byOutput[name]; !ok || better(m, cur) {
				byOutput[name] = m
			}
		}
	}
	methods := make([]planner.Method, 0, len(byOutput))
	for _, m := range byOutput {
		methods = append(methods, m)
	}
	// Map order is random; sort so equal-scoring methods plan the same way
	// every run.
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	if len(methods) == 0 {
		return nil, fmt.Errorf("no tradeable recipes train %s; the planner covers production skills and Woodcutting", skill)
	}
	return methods, nil
}

// playerXP maps lowercase skill names to a player's XP.
func playerXP(ctx context.Context, d *deps, player string) (map[string]int, error) {
	p, err := d.hiscores.Lookup(ctx, player, "regular")
	if err != nil {
		return nil, fmt.Errorf("look up %s: %w", player, err)
	}
	xp := map[string]int{}
	for _, s := range p.Skills {
		xp[strings.ToLower(s.Name)] = int(max(s.XP, 0))
	}
	return xp, nil
}
