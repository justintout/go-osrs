// Package bossing estimates what a boss is worth per hour: loot value,
// hours to its pet, and hours to finish its collection log, from drop
// tables and kill rates.
package bossing

import (
	"math"
	"strings"

	"github.com/justintout/go-osrs/odds"
	"github.com/justintout/go-osrs/wiki"
)

// Boss is one boss to evaluate.
type Boss struct {
	Name         string
	Page         string
	KillsPerHour float64
	// Drops is every drop table line on Page, across all versions.
	Drops []wiki.Drop
	// Section picks a drop table section by its source, such as
	// "Reward Chest (The Gauntlet)#Corrupted". Empty picks the section
	// with the most lines.
	Section string
}

// Result is a boss's estimated value.
type Result struct {
	Name string `json:"name"`
	Page string `json:"page"`
	// Source is the drop table version used, such as "Vorkath#Post-quest".
	Source       string  `json:"source"`
	KillsPerHour float64 `json:"kills_per_hour"`
	GPPerKill    int     `json:"gp_per_kill"`
	GPPerHour    int     `json:"gp_per_hour"`
	Pet          string  `json:"pet,omitempty"`
	// PetChance is per kill. HoursToPet is the mean hours to get it.
	PetChance  float64 `json:"pet_chance,omitempty"`
	HoursToPet float64 `json:"hours_to_pet,omitempty"`
	// LogItems counts collection log items from this boss with a known
	// rate. LogItemsUnknown counts those without one, which HoursToLog
	// leaves out.
	LogItems        int     `json:"log_items"`
	LogItemsUnknown int     `json:"log_items_unknown,omitempty"`
	HoursToLog      float64 `json:"hours_to_log,omitempty"`
}

// Evaluate estimates a boss's value. value returns the coins one unit of
// an item is worth to the player. pets holds pet item names. log is the
// full collection log from wiki.CollectionLog.
//
// When the page has several drop table sections, such as a quest version
// or contract rewards beside the main table, the section with the most
// drop lines is used, since the main table is almost always the longest.
func Evaluate(b Boss, value func(item string) int, pets map[string]bool, log []wiki.LogItem) Result {
	r := Result{Name: b.Name, Page: b.Page, KillsPerHour: b.KillsPerHour}

	bySource := map[string][]wiki.Drop{}
	for _, d := range b.Drops {
		bySource[d.Source] = append(bySource[d.Source], d)
	}
	if _, ok := bySource[b.Section]; ok && b.Section != "" {
		r.Source = b.Section
	} else {
		for source, drops := range bySource {
			n := len(bySource[r.Source])
			if len(drops) > n || (len(drops) == n && source < r.Source) {
				r.Source = source
			}
		}
	}
	r.GPPerKill = int(math.Max(expectedValue(bySource[r.Source], value), 0))
	r.GPPerHour = int(float64(r.GPPerKill) * b.KillsPerHour)

	for _, d := range bySource[r.Source] {
		if pets[d.Item] && d.Probability > 0 {
			r.Pet = d.Item
			r.PetChance = odds.PerKill(d.Probability, max(d.Rolls, 1))
		}
	}
	if r.PetChance > 0 && b.KillsPerHour > 0 {
		r.HoursToPet = odds.ExpectedKills(r.PetChance) / b.KillsPerHour
	}

	_, version, _ := strings.Cut(r.Source, "#")
	var ps []float64
	for _, item := range log {
		s, ok := logSource(item, b.Page, version)
		switch {
		case !ok:
		case s.Probability > 0:
			ps = append(ps, s.Probability)
			r.LogItems++
		default:
			r.LogItemsUnknown++
		}
	}
	if len(ps) > 0 && b.KillsPerHour > 0 {
		r.HoursToLog = odds.ExpectedKillsForAll(ps...) / b.KillsPerHour
	}
	return r
}

// expectedValue is the mean coins per kill from a drop table: each line's
// chance times rolls times average quantity times unit value.
func expectedValue(drops []wiki.Drop, value func(string) int) float64 {
	total := 0.0
	for _, d := range drops {
		qty := float64(d.QuantityLow+d.QuantityHigh) / 2
		total += d.Probability * float64(max(d.Rolls, 1)) * qty * float64(value(d.Item))
	}
	return total
}

// logSource finds an item's source on page, preferring the section that
// matches version and falling back to the page as a whole. Other
// sections, such as contract rewards, are skipped.
func logSource(item wiki.LogItem, page, version string) (wiki.LogSource, bool) {
	var whole *wiki.LogSource
	for i, s := range item.Sources {
		if s.Page != page {
			continue
		}
		if s.Version == version {
			return s, true
		}
		if s.Version == "" && whole == nil {
			whole = &item.Sources[i]
		}
	}
	if whole != nil {
		return *whole, true
	}
	return wiki.LogSource{}, false
}
