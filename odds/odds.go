// Package odds computes drop chances: the chance of a drop within a number
// of kills, how unusual a dry streak is, and the expected kills to collect
// a set of drops.
//
// Probabilities are per kill. For a drop rolled several times per kill,
// convert with PerKill first.
package odds

import "math"

// PerKill converts a per-roll probability into a per-kill probability for
// a table rolled rolls times per kill.
func PerKill(p float64, rolls int) float64 {
	if rolls == 1 {
		// 1-(1-p) loses precision for small p.
		return p
	}
	return 1 - math.Pow(1-p, float64(rolls))
}

// ChanceWithin returns the chance of at least one drop within kills.
func ChanceWithin(p float64, kills int) float64 {
	return 1 - math.Pow(1-p, float64(kills))
}

// DryChance returns the chance of no drop in kills, which is how unusual
// a dry streak of that length is.
func DryChance(p float64, kills int) float64 {
	return math.Pow(1-p, float64(kills))
}

// KillsFor returns the kills needed for a chance of at least one drop. It
// returns 0 for chance <= 0 and +Inf for chance >= 1 or p <= 0.
func KillsFor(p, chance float64) float64 {
	switch {
	case chance <= 0:
		return 0
	case chance >= 1 || p <= 0:
		return math.Inf(1)
	}
	return math.Ceil(math.Log(1-chance) / math.Log(1-p))
}

// ExpectedKills returns the mean kills to get one drop.
func ExpectedKills(p float64) float64 {
	if p <= 0 {
		return math.Inf(1)
	}
	return 1 / p
}

// ExpectedKillsForAll returns the mean kills to get every drop in ps at
// least once, treating drops as independent. It is the sum over n of the
// chance the set is still incomplete after n kills.
func ExpectedKillsForAll(ps ...float64) float64 {
	if len(ps) == 0 {
		return 0
	}
	for _, p := range ps {
		if p <= 0 {
			return math.Inf(1)
		}
	}
	// Stop once the remaining tail is negligible. The rarest drop bounds
	// how slowly the tail shrinks.
	rarest := ps[0]
	for _, p := range ps {
		rarest = min(rarest, p)
	}
	limit := 50 / rarest
	total := 0.0
	for n := 0.0; n < limit; n++ {
		complete := 1.0
		for _, p := range ps {
			complete *= 1 - math.Pow(1-p, n)
		}
		total += 1 - complete
	}
	return total
}
