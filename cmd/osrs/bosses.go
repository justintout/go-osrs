package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/justintout/go-osrs/bossing"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

type bossesIn struct {
	Account string `json:"account,omitempty" jsonschema:"main (default) values loot at live GE prices; ironman values it at high alch"`
	Sort    string `json:"sort,omitempty" jsonschema:"gp (default) for GP per hour, pet for fewest hours to the pet, or log for fewest hours to finish the collection log"`
	Top     int    `json:"top,omitempty" jsonschema:"number of results, 20 by default"`
}

type bossesOut struct {
	Account string           `json:"account"`
	Bosses  []bossing.Result `json:"bosses"`
	// Missing lists bosses with kill rates but no drop table found.
	Missing []string `json:"missing,omitempty"`
	// Raids lists raids left out because their loot depends on points.
	Raids []string `json:"raids_not_modeled,omitempty"`
}

func (o bossesOut) text() string {
	var b strings.Builder
	t := newTable(&b, "BOSS", "KILLS/HR", "GP/KILL", "GP/HR", "PET", "HRS TO PET", "LOG ITEMS", "HRS TO LOG")
	for _, r := range o.Bosses {
		t.row(r.Name, fmt.Sprintf("%g", r.KillsPerHour), num(int64(r.GPPerKill)), num(int64(r.GPPerHour)),
			r.Pet, hours(r.HoursToPet), fmt.Sprint(r.LogItems), hours(r.HoursToLog))
	}
	t.flush()
	b.WriteString("\nGP/hr is loot only, before supplies. Kill rates are Wise Old Man's EHB rates for an efficient " + o.Account + " account.\n")
	if len(o.Raids) > 0 {
		fmt.Fprintf(&b, "not modeled, since raid loot depends on team points: %s\n", strings.Join(o.Raids, ", "))
	}
	if len(o.Missing) > 0 {
		fmt.Fprintf(&b, "no drop table found for: %s\n", strings.Join(o.Missing, ", "))
	}
	return b.String()
}

// bossPageOverrides covers bosses whose loot comes from a reward chest
// page rather than the boss's own page. A "#Section" suffix picks one
// section of that page's drop table.
var bossPageOverrides = map[string]string{
	"barrows_chests":         "Chest (Barrows)",
	"the_gauntlet":           "Reward Chest (The Gauntlet)#Regular",
	"the_corrupted_gauntlet": "Reward Chest (The Gauntlet)#Corrupted",
	"sol_heredit":            "Rewards Chest (Fortis Colosseum)#Wave 12",
}

// unmodeledRaids lists raids whose wiki loot rates are conditional on a
// unique roll and depend on team points, so per-raid value cannot be
// computed from the drop table.
var unmodeledRaids = map[string]bool{
	"chambers_of_xeric": true, "chambers_of_xeric_challenge_mode": true,
	"theatre_of_blood": true, "theatre_of_blood_hard_mode": true,
	"tombs_of_amascut": true, "tombs_of_amascut_expert": true,
}

var bossesCmd = newCommand("bosses", "rank every boss by loot GP per hour, hours to its pet, or hours to finish its collection log, from live prices, wiki drop tables, and Wise Old Man kill rates",
	func(ctx context.Context, d *deps, in bossesIn) (bossesOut, error) {
		account := in.Account
		if account == "" {
			account = "main"
		}
		if account != "main" && account != "ironman" {
			return bossesOut{}, fmt.Errorf("account must be main or ironman")
		}
		rates, err := d.wom.BossRates(ctx, account)
		if err != nil {
			return bossesOut{}, err
		}
		pages, err := bossPages(ctx, d, rates)
		if err != nil {
			return bossesOut{}, err
		}
		out := bossesOut{Account: account}
		cat, latest, _, err := marketData(ctx, d)
		if err != nil {
			return bossesOut{}, err
		}
		petList, err := d.wiki.CategoryMembers(ctx, "Pets")
		if err != nil {
			return bossesOut{}, err
		}
		pets := map[string]bool{}
		for _, p := range petList {
			pets[p] = true
		}
		log, err := d.wiki.CollectionLog(ctx)
		if err != nil {
			return bossesOut{}, err
		}
		for metric := range unmodeledRaids {
			if p, ok := pages[metric]; ok {
				out.Raids = append(out.Raids, p.name)
				delete(pages, metric)
			}
		}
		drops, err := fetchDrops(ctx, d, pages)
		if err != nil {
			return bossesOut{}, err
		}

		value := lootValue(cat, latest, account == "ironman")
		for metric, page := range pages {
			if len(drops[page.page]) == 0 {
				out.Missing = append(out.Missing, page.name)
				continue
			}
			b := bossing.Boss{Name: page.name, Page: page.page, Section: page.section, KillsPerHour: rates[metric], Drops: drops[page.page]}
			out.Bosses = append(out.Bosses, bossing.Evaluate(b, value, pets, log))
		}
		sortBosses(out.Bosses, in.Sort)
		sort.Strings(out.Missing)
		sort.Strings(out.Raids)
		out.Bosses = top(out.Bosses, orDefault(in.Top, 20))
		return out, nil
	})

type bossPage struct{ name, page, section string }

// bossPages maps Wise Old Man boss metrics to display names and wiki
// pages. Display names come from the hiscores, matched by letters and
// digits only, so "kreearra" finds "Kree'Arra".
func bossPages(ctx context.Context, d *deps, rates map[string]float64) (map[string]bossPage, error) {
	topPlayers, err := d.hiscores.SkillLeaderboard(ctx, hiscores.Regular, 0, 1)
	if err != nil {
		return nil, err
	}
	p, err := d.hiscores.Lookup(ctx, topPlayers[0].Name, hiscores.Regular)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, a := range p.Activities {
		names[squash(a.Name)] = a.Name
	}
	pages := map[string]bossPage{}
	var titles []string
	for metric := range rates {
		name, ok := names[squash(metric)]
		if !ok {
			// Hiscores names can be longer, as in "Tombs of Amascut: Expert Mode".
			name = metric
			for k, v := range names {
				if strings.HasPrefix(k, squash(metric)) {
					name = v
				}
			}
		}
		pages[metric] = bossPage{name: name, page: name}
		titles = append(titles, name)
	}
	resolved, err := d.wiki.ResolveTitles(ctx, titles)
	if err != nil {
		return nil, err
	}
	for metric, bp := range pages {
		if o, ok := bossPageOverrides[metric]; ok {
			bp.page, _, _ = strings.Cut(o, "#")
			if strings.Contains(o, "#") {
				bp.section = o
			}
		} else if r, ok := resolved[bp.name]; ok {
			bp.page = r
		}
		pages[metric] = bp
	}
	return pages, nil
}

func squash(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// fetchDrops loads each distinct page's drop table, four at a time.
func fetchDrops(ctx context.Context, d *deps, pages map[string]bossPage) (map[string][]wiki.Drop, error) {
	seen := map[string]bool{}
	var list []string
	for _, p := range pages {
		if !seen[p.page] {
			seen[p.page] = true
			list = append(list, p.page)
		}
	}
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		out      = map[string][]wiki.Drop{}
		sem      = make(chan struct{}, 4)
	)
	for _, page := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			drops, err := d.wiki.Drops(ctx, page)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			out[page] = drops
		}()
	}
	wg.Wait()
	return out, firstErr
}

// lootValue values one unit of an item: the instant-sell price after tax,
// or high alch for ironmen. Coins are worth one each.
func lootValue(cat *prices.Catalog, latest prices.Prices, alch bool) func(string) int {
	return func(name string) int {
		if name == "Coins" {
			return 1
		}
		item, ok := cat.ByName(name)
		if !ok {
			return 0
		}
		if alch {
			return item.HighAlch
		}
		return prices.AfterTax(item.ID, latest[item.ID].Low)
	}
}

func sortBosses(r []bossing.Result, by string) {
	// Bosses with no pet or no known log items sort last.
	inf := func(h float64) float64 {
		if h <= 0 {
			return math.Inf(1)
		}
		return h
	}
	sort.Slice(r, func(i, j int) bool {
		switch by {
		case "pet":
			return inf(r[i].HoursToPet) < inf(r[j].HoursToPet)
		case "log":
			return inf(r[i].HoursToLog) < inf(r[j].HoursToLog)
		default:
			return r[i].GPPerHour > r[j].GPPerHour
		}
	})
}

func hours(h float64) string {
	if h <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f", h)
}
