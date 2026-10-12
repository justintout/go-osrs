package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/profit"
)

const defaultMinVolume = 500

type alchsIn struct {
	MinVolume int  `json:"min_volume,omitempty" jsonschema:"minimum 24h trades for an item to count, 500 by default"`
	Top       int  `json:"top,omitempty" jsonschema:"number of results, 15 by default"`
	F2P       bool `json:"f2p,omitempty" jsonschema:"free-to-play items only"`
}

type alchsOut struct {
	Alchs []profit.Alch `json:"alchs"`
}

func (o alchsOut) text() string {
	var b strings.Builder
	t := newTable(&b, "ITEM", "BUY", "ALCH", "PROFIT", "LIMIT", "PROFIT/LIMIT", "24H VOL")
	for _, a := range o.Alchs {
		t.row(a.Name, num(int64(a.BuyPrice)), num(int64(a.HighAlch)), num(int64(a.Profit)), num(int64(a.BuyLimit)), num(int64(a.ProfitPerLimit)), num(int64(a.Volume24h)))
	}
	t.flush()
	return b.String()
}

var alchsCmd = newCommand("alchs", "list the most profitable items to buy and high alch at live prices, after nature rune cost",
	func(ctx context.Context, d *deps, in alchsIn) (alchsOut, error) {
		cat, latest, vols, err := marketData(ctx, d)
		if err != nil {
			return alchsOut{}, err
		}
		var out []profit.Alch
		for _, a := range profit.Alchs(cat.Items(), latest, vols, orDefault(in.MinVolume, defaultMinVolume)) {
			if !in.F2P || !a.Members {
				out = append(out, a)
			}
		}
		return alchsOut{Alchs: top(out, orDefault(in.Top, 15))}, nil
	})

type recipesIn struct {
	Player    string `json:"player,omitempty" arg:"" jsonschema:"only recipes this player has the levels for"`
	Skill     string `json:"skill,omitempty" jsonschema:"only recipes that use this skill"`
	MinVolume int    `json:"min_volume,omitempty" jsonschema:"minimum 24h trades for the output and each material, 500 by default"`
	Top       int    `json:"top,omitempty" jsonschema:"number of results, 15 by default"`
	F2P       bool   `json:"f2p,omitempty" jsonschema:"free-to-play recipes only"`
}

type recipeLine struct {
	Output          string   `json:"output"`
	Page            string   `json:"page"`
	Materials       []string `json:"materials"`
	Skills          []string `json:"skills"`
	Cost            int      `json:"cost"`
	Revenue         int      `json:"revenue"`
	Profit          int      `json:"profit"`
	ProfitPerHour   int      `json:"profit_per_hour,omitempty"`
	OutputVolume24h int      `json:"output_volume_24h"`
}

type recipesOut struct {
	Recipes []recipeLine `json:"recipes"`
}

func (o recipesOut) text() string {
	var b strings.Builder
	t := newTable(&b, "MAKE", "FROM", "NEEDS", "COST", "PROFIT", "PROFIT/HR", "24H VOL")
	for _, r := range o.Recipes {
		perHour := "-"
		if r.ProfitPerHour > 0 {
			perHour = num(int64(r.ProfitPerHour))
		}
		t.row(r.Output, strings.Join(r.Materials, ", "), strings.Join(r.Skills, ", "), num(int64(r.Cost)), num(int64(r.Profit)), perHour, num(int64(r.OutputVolume24h)))
	}
	t.flush()
	b.WriteString("\nprofit/hr assumes continuous actions with no banking and ignores buy limits\n")
	return b.String()
}

var recipesCmd = newCommand("recipes", "list profitable item recipes at live prices, optionally only those a player has the levels for",
	func(ctx context.Context, d *deps, in recipesIn) (recipesOut, error) {
		var levels map[string]int
		if in.Player != "" {
			p, err := d.hiscores.Lookup(ctx, in.Player, hiscores.Regular)
			if errors.Is(err, osrs.ErrNotFound) {
				return recipesOut{}, fmt.Errorf("%s is not on the hiscores", in.Player)
			}
			if err != nil {
				return recipesOut{}, err
			}
			levels = map[string]int{}
			for _, s := range p.Skills {
				levels[strings.ToLower(s.Name)] = s.Level
			}
		}
		recipes, err := d.wiki.Recipes(ctx)
		if err != nil {
			return recipesOut{}, err
		}
		cat, latest, vols, err := marketData(ctx, d)
		if err != nil {
			return recipesOut{}, err
		}
		var out []recipeLine
		seen := map[string]bool{}
	recipe:
		for _, rp := range profit.Recipes(recipes, cat, latest, vols, orDefault(in.MinVolume, defaultMinVolume)) {
			r := rp.Recipe
			if in.F2P && r.Members {
				continue
			}
			usesSkill := in.Skill == ""
			line := recipeLine{Output: r.Output.Name, Page: r.Page, Cost: rp.Cost, Revenue: rp.Revenue, Profit: rp.Profit, ProfitPerHour: rp.ProfitPerHour, OutputVolume24h: rp.OutputVolume24h}
			for _, s := range r.Skills {
				if levels != nil && levels[strings.ToLower(s.Name)] < s.Level {
					continue recipe
				}
				usesSkill = usesSkill || strings.EqualFold(s.Name, in.Skill)
				line.Skills = append(line.Skills, fmt.Sprintf("%d %s", s.Level, s.Name))
			}
			// Results are sorted best first, so the first recipe per output wins.
			if !usesSkill || seen[r.Output.Name] {
				continue
			}
			seen[r.Output.Name] = true
			for _, m := range r.Materials {
				line.Materials = append(line.Materials, fmt.Sprintf("%gx %s", m.Quantity, m.Name))
			}
			out = append(out, line)
		}
		return recipesOut{Recipes: top(out, orDefault(in.Top, 15))}, nil
	})

func top[T any](s []T, n int) []T {
	return s[:min(len(s), n)]
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
