package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/justintout/go-osrs/planner"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/skills"
	"github.com/justintout/go-osrs/wiki"
	"github.com/justintout/go-osrs/wom"
	"github.com/justintout/go-osrs/woodcutting"
)

type trainIn struct {
	Skill      string  `json:"skill" arg:"" jsonschema:"skill to train, such as Fletching or Agility"`
	Player     string  `json:"player,omitempty" jsonschema:"start from this player's current XP"`
	From       int     `json:"from,omitempty" jsonschema:"start level when no player is given, 1 by default"`
	To         int     `json:"to,omitempty" jsonschema:"target level, 99 by default"`
	By         string  `json:"by,omitempty" jsonschema:"what to minimize: hours (default), coins, actions, or balanced (hours plus coins valued at gp_per_hour)"`
	GPPerHour  float64 `json:"gp_per_hour,omitempty" jsonschema:"value of an hour of play in coins, required for balanced"`
	Account    string  `json:"account,omitempty" jsonschema:"main (default) or ironman, for efficient training rates"`
	MinVolume  int     `json:"min_volume,omitempty" jsonschema:"minimum 24h trades for each recipe input, and for the output if it is sold, 500 by default"`
	MarketDays int     `json:"market_days,omitempty" jsonschema:"cap each recipe at this many days of its items' whole-market volume, 1 by default"`
	Exclude    string  `json:"exclude,omitempty" jsonschema:"comma-separated parts of method names to leave out, such as one-off builds the wiki lists as recipes"`
}

type trainOut struct {
	Skill     string       `json:"skill"`
	FromXP    int          `json:"from_xp"`
	ToXP      int          `json:"to_xp"`
	By        string       `json:"by"`
	GPPerHour float64      `json:"gp_per_hour,omitempty"`
	Plan      planner.Plan `json:"plan"`
	Notes     []string     `json:"notes"`
}

func (o trainOut) text() string {
	var b strings.Builder
	goal := o.By
	if o.By == "balanced" {
		goal = fmt.Sprintf("hours plus coins at %s gp per hour", num(int64(o.GPPerHour)))
	}
	fmt.Fprintf(&b, "%s level %d to %d, minimizing %s\n\n", o.Skill, skills.Level(o.FromXP), skills.Level(o.ToXP), goal)
	t := newTable(&b, "LEVELS", "METHOD", "XP/HR", "HOURS", "ACTIONS", "GP")
	for _, s := range o.Plan.Steps {
		rate, hours, actions := "-", "-", "-"
		if r := rateAtStart(s); r > 0 {
			rate = num(int64(r))
		}
		if s.Hours > 0 {
			hours = fmt.Sprintf("%.1f", s.Hours)
		}
		if s.Actions > 0 {
			actions = num(int64(s.Actions))
		}
		t.row(fmt.Sprintf("%d-%d", skills.Level(s.FromXP), skills.Level(s.ToXP)), s.Method.Name, rate, hours, actions, signed(s.GP))
	}
	t.flush()
	fmt.Fprintf(&b, "\ntotal")
	if o.Plan.Hours > 0 {
		fmt.Fprintf(&b, "  %.1f hours", o.Plan.Hours)
	}
	if o.Plan.Actions > 0 {
		fmt.Fprintf(&b, "  %s actions", num(int64(o.Plan.Actions)))
	}
	fmt.Fprintf(&b, "  %s gp\n", signed(o.Plan.GP))
	for _, n := range o.Notes {
		b.WriteString(n + "\n")
	}
	return b.String()
}

func rateAtStart(s planner.Step) float64 {
	r := 0.0
	for _, rt := range s.Method.Rates {
		if rt.FromXP <= s.FromXP {
			r = rt.XPPerHour
		}
	}
	return r
}

var trainCmd = newCommand("train", "plan training a skill to a target, rebalancing at every unlock and rate breakpoint, minimizing hours, coins, actions, or hours plus coins valued at your gp per hour",
	func(ctx context.Context, d *deps, in trainIn) (trainOut, error) {
		if err := need("skill", in.Skill); err != nil {
			return trainOut{}, err
		}
		_, skill, err := skillID(in.Skill)
		if err != nil || skill == "Overall" {
			return trainOut{}, fmt.Errorf("unknown skill %q", in.Skill)
		}
		goal := planner.Goal{GPPerHour: in.GPPerHour}
		by := orDefaultString(in.By, "hours")
		switch by {
		case "hours":
			goal.Objective = planner.Hours
		case "coins":
			goal.Objective = planner.Coins
		case "actions":
			goal.Objective = planner.Actions
		case "balanced":
			goal.Objective = planner.Balanced
			if in.GPPerHour <= 0 {
				return trainOut{}, fmt.Errorf("balanced needs gp_per_hour, the coins an hour of your time is worth")
			}
		default:
			return trainOut{}, fmt.Errorf("by must be hours, coins, actions, or balanced")
		}

		fromXP := skills.XPForLevel(max(in.From, 1))
		if in.Player != "" {
			xp, err := playerXP(ctx, d, in.Player)
			if err != nil {
				return trainOut{}, err
			}
			fromXP = xp[strings.ToLower(skill)]
		}
		toXP := skills.XPForLevel(orDefault(in.To, 99))
		if fromXP >= toXP {
			return trainOut{}, fmt.Errorf("already at or past level %d", orDefault(in.To, 99))
		}

		methods, notes, err := trainingMethods(ctx, d, skill, orDefaultString(in.Account, "main"), orDefault(in.MinVolume, defaultMinVolume), orDefault(in.MarketDays, 1))
		if err != nil {
			return trainOut{}, err
		}
		if in.Exclude != "" {
			var kept []planner.Method
		method:
			for _, m := range methods {
				for _, x := range strings.Split(in.Exclude, ",") {
					if x = strings.TrimSpace(x); x != "" && strings.Contains(strings.ToLower(m.Name), strings.ToLower(x)) {
						continue method
					}
				}
				kept = append(kept, m)
			}
			methods = kept
		}
		plan, err := planner.Solve(methods, fromXP, toXP, goal)
		if err != nil {
			return trainOut{}, fmt.Errorf("%s: %w", skill, err)
		}
		return trainOut{Skill: skill, FromXP: fromXP, ToXP: toXP, By: by, GPPerHour: in.GPPerHour, Plan: plan, Notes: notes}, nil
	})

// trainingMethods gathers ways to train skill from three sources:
//
//   - Wise Old Man's efficient training rates, which give XP per hour for
//     most skills but no per-action XP or costs, so they count toward
//     hours only.
//   - Wiki recipes, priced at live GE prices: inputs are bought, and the
//     output is sold if it trades. Rates come from each recipe's game
//     ticks with no banking, capped at the efficient rate where Wise Old
//     Man has one. Each recipe is capped at market_days of its items'
//     volume, since a plan that floods the market cannot trade at the
//     quoted prices.
//   - The woodcutting tree table, for actions.
func trainingMethods(ctx context.Context, d *deps, skill, account string, minVolume, marketDays int) ([]planner.Method, []string, error) {
	var methods []planner.Method
	var notes []string

	womRates, err := d.wom.SkillRates(ctx, account)
	if err != nil {
		return nil, nil, err
	}
	efficient := womRates[strings.ToLower(skill)]
	if skill == string(skills.Runecraft) {
		efficient = womRates["runecrafting"]
	}
	if len(efficient) > 0 {
		methods = append(methods, efficientMethods(efficient)...)
		notes = append(notes, "efficient rates are Wise Old Man's EHP rates for a "+account+" account, which assume skilled, attentive play")
	}

	recipes, err := d.wiki.Recipes(ctx)
	if err != nil {
		return nil, nil, err
	}
	cat, latest, vols, err := marketData(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	recipeMethods := recipeTraining(recipes, cat, latest, vols, skill, efficient, minVolume, marketDays)
	if len(recipeMethods) > 0 {
		methods = append(methods, recipeMethods...)
		if len(efficient) > 0 {
			notes = append(notes, "recipe rates come from game ticks with no banking, capped at the efficient rate; outputs that cannot be sold count as worth nothing")
		} else {
			notes = append(notes, "recipe rates come from game ticks with no banking; Wise Old Man has no efficient rate for "+skill+" to cap them, so treat hours as a lower bound")
		}
	}
	if skill == string(skills.Woodcutting) {
		methods = append(methods, woodcutting.Methods()...)
	}
	if len(methods) == 0 {
		return nil, nil, fmt.Errorf("no training data for %s", skill)
	}
	return methods, notes, nil
}

// efficientMethods turns a Wise Old Man rate schedule into one method per
// description, each unlocked where it first appears and keeping its rates
// as levels rise.
func efficientMethods(rates []wom.TrainingRate) []planner.Method {
	byName := map[string]*planner.Method{}
	var order []string
	for _, r := range rates {
		m, ok := byName[r.Description]
		if !ok {
			m = &planner.Method{Name: r.Description, Level: skills.Level(r.StartXP)}
			byName[r.Description] = m
			order = append(order, r.Description)
		}
		m.Rates = append(m.Rates, planner.Rate{FromXP: r.StartXP, XPPerHour: r.XPPerHour})
	}
	out := make([]planner.Method, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}

func recipeTraining(recipes []wiki.Recipe, cat *prices.Catalog, latest prices.Prices, vols map[int]int, skill string, efficient []wom.TrainingRate, minVolume, marketDays int) []planner.Method {
	best := map[string]planner.Method{}
	for _, r := range recipes {
		var rs *wiki.RecipeSkill
		for i, s := range r.Skills {
			if strings.EqualFold(s.Name, skill) && s.XP > 0 {
				rs = &r.Skills[i]
			}
		}
		// Recipes without tick timings are mostly one-time builds, such as
		// quest or camp construction; STASH units are built once each.
		if rs == nil || len(r.Materials) == 0 || r.Ticks <= 0 || strings.Contains(r.Output.Name, "STASH") {
			continue
		}
		cost, priced := 0.0, true
		maxActions := math.MaxInt
		for _, m := range r.Materials {
			item, ok := cat.ByName(m.Name)
			buy := latest[item.ID].High
			if !ok || buy == 0 || vols[item.ID] < minVolume || m.Quantity <= 0 {
				priced = false
				break
			}
			cost += float64(buy) * m.Quantity
			maxActions = min(maxActions, int(float64(vols[item.ID]*marketDays)/m.Quantity))
		}
		if !priced {
			continue
		}
		revenue := 0.0
		if out, ok := cat.ByName(r.Output.Name); ok && latest[out.ID].Low > 0 && vols[out.ID] >= minVolume {
			revenue = float64(prices.AfterTax(out.ID, latest[out.ID].Low)) * r.Output.Quantity
			maxActions = min(maxActions, int(float64(vols[out.ID]*marketDays)/r.Output.Quantity))
		}
		m := planner.Method{Name: r.Output.Name, Level: rs.Level, XP: rs.XP, GP: revenue - cost, MaxActions: maxActions,
			Rates: cappedRates(rs.XP*3600/(r.Ticks*0.6), efficient)}
		// Recipes that make the same item share its market; keep the
		// cheapest per XP.
		if cur, ok := best[m.Name]; !ok || m.GP/m.XP > cur.GP/cur.XP {
			best[m.Name] = m
		}
	}
	out := make([]planner.Method, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// cappedRates holds a recipe's tick rate to the efficient rate at each of
// the efficient schedule's breakpoints.
func cappedRates(tickRate float64, efficient []wom.TrainingRate) []planner.Rate {
	if len(efficient) == 0 {
		return []planner.Rate{{FromXP: 0, XPPerHour: tickRate}}
	}
	rates := []planner.Rate{{FromXP: 0, XPPerHour: math.Min(tickRate, efficient[0].XPPerHour)}}
	for _, e := range efficient {
		rates = append(rates, planner.Rate{FromXP: e.StartXP, XPPerHour: math.Min(tickRate, e.XPPerHour)})
	}
	return rates
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
