package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/justintout/go-osrs/odds"
	"github.com/justintout/go-osrs/wiki"
)

type oddsIn struct {
	Monster string `json:"monster,omitempty" arg:"" jsonschema:"monster wiki page whose drop table to use"`
	Items   string `json:"items,omitempty" jsonschema:"comma-separated drop names, such as Vorki,Draconic visage"`
	Rate    string `json:"rate,omitempty" jsonschema:"a drop rate such as 1/3000, instead of monster and items"`
	Kills   int    `json:"kills,omitempty" jsonschema:"kill count to evaluate: chance of a drop by now and how unusual a dry streak this is"`
}

type oddsItem struct {
	Item          string  `json:"item"`
	Source        string  `json:"source,omitempty"`
	Rarity        string  `json:"rarity"`
	PerKill       float64 `json:"per_kill"`
	ExpectedKills float64 `json:"expected_kills"`
	Kills50       float64 `json:"kills_for_50pct"`
	Kills90       float64 `json:"kills_for_90pct"`
	Kills99       float64 `json:"kills_for_99pct"`
	ChanceByKills float64 `json:"chance_by_kills,omitempty"`
	DryChance     float64 `json:"dry_chance,omitempty"`
}

type oddsOut struct {
	Kills int        `json:"kills,omitempty"`
	Items []oddsItem `json:"items"`
	// ExpectedKillsForAll is the mean kills to get every item, when there
	// is more than one.
	ExpectedKillsForAll float64 `json:"expected_kills_for_all,omitempty"`
}

func (o oddsOut) text() string {
	var b strings.Builder
	headers := []string{"ITEM", "RARITY", "EXPECTED", "50%", "90%", "99%"}
	if o.Kills > 0 {
		headers = append(headers, fmt.Sprintf("BY %s KC", num(int64(o.Kills))), "DRY THIS LONG")
	}
	t := newTable(&b, headers...)
	for _, i := range o.Items {
		row := []string{i.Item, i.Rarity, kills(i.ExpectedKills), kills(i.Kills50), kills(i.Kills90), kills(i.Kills99)}
		if o.Kills > 0 {
			row = append(row, pct(i.ChanceByKills), pct(i.DryChance))
		}
		t.row(row...)
	}
	t.flush()
	if o.ExpectedKillsForAll > 0 {
		fmt.Fprintf(&b, "\nexpected kills for all %d: %s\n", len(o.Items), kills(o.ExpectedKillsForAll))
	}
	return b.String()
}

var oddsCmd = newCommand("odds", "drop odds: expected kills, kills for 50/90/99% chance, chance by a kill count, dry-streak rarity, and kills to collect a set",
	func(ctx context.Context, d *deps, in oddsIn) (oddsOut, error) {
		var items []oddsItem
		switch {
		case in.Rate != "":
			p, ok := wiki.ParseRarity(in.Rate)
			if !ok || p <= 0 || p > 1 {
				return oddsOut{}, fmt.Errorf("rate %q is not a probability like 1/3000", in.Rate)
			}
			items = append(items, oddsItem{Item: in.Rate, Rarity: in.Rate, PerKill: p})
		case in.Monster != "" && in.Items != "":
			drops, err := d.wiki.Drops(ctx, in.Monster)
			if err != nil {
				return oddsOut{}, err
			}
			for _, name := range strings.Split(in.Items, ",") {
				item, err := bestDrop(drops, strings.TrimSpace(name), in.Monster)
				if err != nil {
					return oddsOut{}, err
				}
				items = append(items, item)
			}
		default:
			return oddsOut{}, fmt.Errorf("set rate, or monster and items")
		}

		out := oddsOut{Kills: in.Kills}
		var ps []float64
		for i := range items {
			it := &items[i]
			it.ExpectedKills = odds.ExpectedKills(it.PerKill)
			it.Kills50, it.Kills90, it.Kills99 = odds.KillsFor(it.PerKill, 0.5), odds.KillsFor(it.PerKill, 0.9), odds.KillsFor(it.PerKill, 0.99)
			if in.Kills > 0 {
				it.ChanceByKills, it.DryChance = odds.ChanceWithin(it.PerKill, in.Kills), odds.DryChance(it.PerKill, in.Kills)
			}
			ps = append(ps, it.PerKill)
		}
		out.Items = items
		if len(ps) > 1 {
			out.ExpectedKillsForAll = odds.ExpectedKillsForAll(ps...)
		}
		return out, nil
	})

// bestDrop picks the most likely table line for an item. A monster page
// can list the same item for several versions or quantities.
func bestDrop(drops []wiki.Drop, item, monster string) (oddsItem, error) {
	var best *wiki.Drop
	for i, d := range drops {
		if strings.EqualFold(d.Item, item) && d.Probability > 0 {
			p := odds.PerKill(d.Probability, max(d.Rolls, 1))
			if best == nil || p > odds.PerKill(best.Probability, max(best.Rolls, 1)) {
				best = &drops[i]
			}
		}
	}
	if best == nil {
		return oddsItem{}, fmt.Errorf("%s has no drop named %q with a known rate", monster, item)
	}
	return oddsItem{Item: best.Item, Source: best.Source, Rarity: best.Rarity, PerKill: odds.PerKill(best.Probability, max(best.Rolls, 1))}, nil
}

func kills(k float64) string {
	if math.IsInf(k, 1) {
		return "never"
	}
	return num(int64(math.Round(k)))
}

func pct(p float64) string { return fmt.Sprintf("%.1f%%", p*100) }
