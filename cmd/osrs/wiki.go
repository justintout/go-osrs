package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/justintout/go-osrs/wiki"
)

type pageIn struct {
	Page string `json:"page" arg:"" jsonschema:"wiki page title, such as Abyssal whip or Vorkath"`
}

type wikiItemOut struct {
	Items   []wiki.Item    `json:"items"`
	Bonuses []wiki.Bonuses `json:"bonuses,omitempty"`
}

func (o wikiItemOut) text() string {
	var b strings.Builder
	for _, i := range o.Items {
		name := i.Name
		if i.Version != "" {
			name += " (" + i.Version + ")"
		}
		fmt.Fprintf(&b, "%s  ids %v\n  %s\n  members %v  tradeable %v  value %s  high alch %s  weight %g kg  buy limit %d\n",
			name, i.IDs(), i.Examine, bool(i.Members), bool(i.Tradeable), num(int64(i.Value)), num(int64(i.HighAlch)), i.Weight, i.BuyLimit)
	}
	for _, x := range o.Bonuses {
		fmt.Fprintf(&b, "\n%s slot  attack stab %+d slash %+d crush %+d magic %+d ranged %+d\n  defence stab %+d slash %+d crush %+d magic %+d ranged %+d\n  strength %+d  ranged str %+d  magic dmg %+g%%  prayer %+d",
			x.Slot, x.StabAttack, x.SlashAttack, x.CrushAttack, x.MagicAttack, x.RangedAttack,
			x.StabDefence, x.SlashDefence, x.CrushDefence, x.MagicDefence, x.RangedDefence,
			x.Strength, x.RangedStr, x.MagicDamage, x.Prayer)
		if x.AttackSpeed > 0 {
			fmt.Fprintf(&b, "  speed %d", x.AttackSpeed)
		}
		b.WriteString("\n")
	}
	return b.String()
}

var wikiItemCmd = newCommand("wiki-item", "get an item's wiki infobox: examine, value, alch, weight, buy limit, and equipment bonuses",
	func(ctx context.Context, d *deps, in pageIn) (wikiItemOut, error) {
		if err := need("page", in.Page); err != nil {
			return wikiItemOut{}, err
		}
		items, err := d.wiki.Item(ctx, in.Page)
		if err != nil {
			return wikiItemOut{}, err
		}
		if len(items) == 0 {
			return wikiItemOut{}, fmt.Errorf("no item infobox on wiki page %q", in.Page)
		}
		bonuses, err := d.wiki.Bonuses(ctx, in.Page)
		return wikiItemOut{Items: items, Bonuses: bonuses}, err
	})

type wikiMonsterOut struct {
	Versions []wiki.Monster `json:"versions"`
}

func (o wikiMonsterOut) text() string {
	var b strings.Builder
	for _, m := range o.Versions {
		name := m.Name
		if m.Version != "" {
			name += " (" + m.Version + ")"
		}
		fmt.Fprintf(&b, "%s  combat %d  hitpoints %d  attributes %s\n  max hit %s\n", name, m.CombatLevel, m.Hitpoints, strings.Join(m.Attributes, ", "), strings.Join(m.MaxHit, ", "))
		fmt.Fprintf(&b, "  levels att %d str %d def %d ranged %d magic %d\n  defence stab %+d slash %+d crush %+d magic %+d ranged %+d\n",
			m.AttackLevel, m.StrengthLevel, m.DefenceLevel, m.RangedLevel, m.MagicLevel, m.StabDefence, m.SlashDefence, m.CrushDefence, m.MagicDefence, m.RangedDefence)
		if m.SlayerXP > 0 {
			fmt.Fprintf(&b, "  slayer level %d  xp %g  categories %s\n", m.SlayerLevel, m.SlayerXP, strings.Join(m.SlayerCategory, ", "))
		}
	}
	return b.String()
}

var wikiMonsterCmd = newCommand("wiki-monster", "get a monster's wiki stats: combat level, hitpoints, max hits, defences, and slayer info",
	func(ctx context.Context, d *deps, in pageIn) (wikiMonsterOut, error) {
		if err := need("page", in.Page); err != nil {
			return wikiMonsterOut{}, err
		}
		m, err := d.wiki.Monster(ctx, in.Page)
		if err == nil && len(m) == 0 {
			err = fmt.Errorf("no monster infobox on wiki page %q", in.Page)
		}
		return wikiMonsterOut{Versions: m}, err
	})

type dropsIn struct {
	Monster string `json:"monster,omitempty" arg:"" jsonschema:"monster wiki page, to list its drops"`
	Item    string `json:"item,omitempty" jsonschema:"item name, to list what drops it"`
}

type dropsOut struct {
	Drops []wiki.Drop `json:"drops"`
}

func (o dropsOut) text() string {
	var b strings.Builder
	t := newTable(&b, "ITEM", "SOURCE", "QUANTITY", "RARITY", "PROBABILITY")
	for _, d := range o.Drops {
		qty := strconv.Itoa(d.QuantityLow)
		if d.QuantityHigh != d.QuantityLow {
			qty += "-" + strconv.Itoa(d.QuantityHigh)
		}
		p := "-"
		if d.Probability > 0 {
			p = strconv.FormatFloat(d.Probability, 'g', 4, 64)
		}
		t.row(d.Item, d.Source, qty, d.Rarity, p)
	}
	t.flush()
	return b.String()
}

var dropsCmd = newCommand("drops", "list a monster's drop table, or every source of an item, with drop probabilities",
	func(ctx context.Context, d *deps, in dropsIn) (dropsOut, error) {
		if (in.Monster == "") == (in.Item == "") {
			return dropsOut{}, fmt.Errorf("set exactly one of monster or item")
		}
		var drops []wiki.Drop
		var err error
		if in.Monster != "" {
			drops, err = d.wiki.Drops(ctx, in.Monster)
		} else {
			drops, err = d.wiki.DropSources(ctx, in.Item)
		}
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].Probability > drops[j].Probability })
		return dropsOut{Drops: drops}, err
	})

type wikiSearchIn struct {
	Query string `json:"query" arg:"" jsonschema:"search text"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum results, 5 by default"`
}

type wikiSearchOut struct {
	Results []wiki.SearchResult `json:"results"`
}

func (o wikiSearchOut) text() string {
	var b strings.Builder
	for _, r := range o.Results {
		fmt.Fprintf(&b, "%s\n  %s\n", r.Title, r.Snippet)
	}
	return b.String()
}

var wikiSearchCmd = newCommand("wiki-search", "search OSRS Wiki pages",
	func(ctx context.Context, d *deps, in wikiSearchIn) (wikiSearchOut, error) {
		if err := need("query", in.Query); err != nil {
			return wikiSearchOut{}, err
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 5
		}
		r, err := d.wiki.Search(ctx, in.Query, limit)
		return wikiSearchOut{Results: r}, err
	})

type wikiSummaryOut struct {
	Page    string `json:"page"`
	Summary string `json:"summary"`
}

func (o wikiSummaryOut) text() string { return o.Summary + "\n" }

var wikiSummaryCmd = newCommand("wiki-summary", "get the plain-text introduction of an OSRS Wiki page",
	func(ctx context.Context, d *deps, in pageIn) (wikiSummaryOut, error) {
		if err := need("page", in.Page); err != nil {
			return wikiSummaryOut{}, err
		}
		s, err := d.wiki.Summary(ctx, in.Page)
		if err == nil && s == "" {
			err = fmt.Errorf("no wiki page %q", in.Page)
		}
		return wikiSummaryOut{Page: in.Page, Summary: s}, err
	})
