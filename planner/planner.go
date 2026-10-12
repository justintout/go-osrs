// Package planner plans skill training from a start XP to a target. It
// splits the range at every breakpoint, where a method unlocks or a
// method's XP rate changes, and assigns XP to methods to minimize the
// chosen cost: actions, coins, hours, or hours plus coins valued at a
// rate. Plans rebalance at each breakpoint instead of fixing one method
// for the whole goal.
package planner

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/justintout/go-osrs/skills"
)

// Method is one way to train a skill.
type Method struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
	// XP is gained per action, or 0 when only an hourly rate is known.
	XP float64 `json:"xp,omitempty"`
	// GP is the coins gained per action. A negative GP is a cost.
	GP float64 `json:"gp"`
	// Rates gives XP per hour by XP, so a method can speed up as levels
	// raise its success rate. The rate at FromXP holds until the next
	// entry. Empty means the hourly rate is unknown.
	Rates []Rate `json:"rates,omitempty"`
	// MaxActions caps uses, such as how much output the market absorbs.
	// 0 means no cap. It requires XP.
	MaxActions int `json:"max_actions,omitempty"`
}

// Rate is a method's XP per hour from FromXP on.
type Rate struct {
	FromXP    int     `json:"from_xp"`
	XPPerHour float64 `json:"xp_per_hour"`
}

func (m Method) rateAt(xp int) float64 {
	r := 0.0
	for _, rt := range m.Rates {
		if rt.FromXP <= xp {
			r = rt.XPPerHour
		}
	}
	return r
}

// Objective is what a plan minimizes.
type Objective int

const (
	// Actions minimizes the number of actions. Methods need XP.
	Actions Objective = iota
	// Coins minimizes coins spent, or maximizes coins made. Methods
	// need XP.
	Coins
	// Hours minimizes training time. Methods need Rates.
	Hours
	// Balanced minimizes hours plus coins spent divided by the goal's
	// GPPerHour, the value of an hour of play. Methods need XP and Rates.
	Balanced
)

// Goal sets the objective.
type Goal struct {
	Objective Objective
	// GPPerHour values time for Balanced.
	GPPerHour float64
}

// costPerXP is the method's cost for each XP gained at xp, and whether the
// method can be scored for the goal.
func (g Goal) costPerXP(m Method, xp int) (float64, bool) {
	rate := m.rateAt(xp)
	switch g.Objective {
	case Actions:
		return 1 / m.XP, m.XP > 0
	case Coins:
		return -m.GP / m.XP, m.XP > 0
	case Hours:
		return 1 / rate, rate > 0
	case Balanced:
		return 1/rate - m.GP/m.XP/g.GPPerHour, rate > 0 && m.XP > 0 && g.GPPerHour > 0
	}
	return 0, false
}

// Step is a stretch of training with one method.
type Step struct {
	Method Method `json:"method"`
	FromXP int    `json:"from_xp"`
	ToXP   int    `json:"to_xp"`
	// Actions is set when the method's XP per action is known.
	Actions int `json:"actions,omitempty"`
	// GP is coins gained over the step; negative is a cost.
	GP int `json:"gp"`
	// Hours is set when the method's rate is known.
	Hours float64 `json:"hours,omitempty"`
}

// Plan is a full training plan.
type Plan struct {
	Steps   []Step  `json:"steps"`
	Actions int     `json:"actions,omitempty"`
	GP      int     `json:"gp"`
	Hours   float64 `json:"hours,omitempty"`
}

type segment struct{ from, to int }

// Solve plans training from fromXP to toXP. Methods that cannot be scored
// for the goal are ignored.
//
// The XP range is split at every method's unlock level and every rate
// change. Each segment takes its cheapest available method; capped
// methods compete through a minimum-cost flow, so a capped method's XP is
// spent where it saves the most.
func Solve(methods []Method, fromXP, toXP int, goal Goal) (Plan, error) {
	if fromXP < 0 || toXP > skills.MaxXP || fromXP >= toXP {
		return Plan{}, fmt.Errorf("planner: xp range %d to %d must rise within 0 to %d", fromXP, toXP, skills.MaxXP)
	}
	segs := segments(methods, fromXP, toXP)

	// cost[m][s] is the cost per XP of method m in segment s, or NaN when
	// the method is locked or unscorable there.
	var usable []Method
	var cost [][]float64
	for _, m := range methods {
		row := make([]float64, len(segs))
		any := false
		for s, sg := range segs {
			c, ok := goal.costPerXP(m, sg.from)
			if !ok || skills.Level(sg.from) < m.Level {
				row[s] = math.NaN()
				continue
			}
			row[s], any = c, true
		}
		if any {
			usable = append(usable, m)
			cost = append(cost, row)
		}
	}
	if len(usable) == 0 {
		return Plan{}, errors.New("planner: no method can be scored for this goal")
	}
	for s, sg := range segs {
		covered := false
		for m := range usable {
			if !math.IsNaN(cost[m][s]) {
				covered = true
			}
		}
		if !covered {
			return Plan{}, fmt.Errorf("planner: no usable method at level %d", skills.Level(sg.from))
		}
	}

	xp, unmet := assign(usable, segs, cost)
	if unmet > 0.5 {
		return Plan{}, fmt.Errorf("planner: method caps leave %.0f XP unplanned", unmet)
	}
	return build(usable, segs, xp), nil
}

// segments splits [from, to) at unlock levels and rate changes.
func segments(methods []Method, from, to int) []segment {
	cuts := map[int]bool{from: true, to: true}
	for _, m := range methods {
		cuts[skills.XPForLevel(m.Level)] = true
		for _, r := range m.Rates {
			cuts[r.FromXP] = true
		}
	}
	var pts []int
	for c := range cuts {
		if c >= from && c <= to {
			pts = append(pts, c)
		}
	}
	sort.Ints(pts)
	segs := make([]segment, 0, len(pts)-1)
	for i := 0; i+1 < len(pts); i++ {
		segs = append(segs, segment{pts[i], pts[i+1]})
	}
	return segs
}

// build turns per-segment XP assignments into ordered, merged steps.
func build(methods []Method, segs []segment, assign [][]float64) Plan {
	var p Plan
	type piece struct {
		m        int
		from, to int
	}
	var pieces []piece
	last := -1 // method that ended the previous segment
	for s, sg := range segs {
		// When a segment splits between methods, start with the one that
		// ended the previous segment and finish with one that continues
		// into the next, so steps merge instead of flipping back and forth.
		var used []int
		for m := range methods {
			if assign[m][s] > 0.5 {
				used = append(used, m)
			}
		}
		rank := func(m int) int {
			switch {
			case m == last:
				return 0
			case s+1 < len(segs) && assign[m][s+1] > 0.5:
				return 2
			}
			return 1
		}
		sort.SliceStable(used, func(i, j int) bool { return rank(used[i]) < rank(used[j]) })
		at := float64(sg.from)
		for _, m := range used {
			x := assign[m][s]
			pieces = append(pieces, piece{m, int(math.Round(at)), int(math.Round(at + x))})
			at += x
			last = m
		}
	}
	for _, pc := range pieces {
		m := methods[pc.m]
		n := len(p.Steps)
		if n > 0 && p.Steps[n-1].Method.Name == m.Name && p.Steps[n-1].ToXP == pc.from {
			p.Steps[n-1].ToXP = pc.to
		} else {
			p.Steps = append(p.Steps, Step{Method: m, FromXP: pc.from, ToXP: pc.to})
		}
	}
	for i := range p.Steps {
		st := &p.Steps[i]
		m := st.Method
		if m.XP > 0 {
			st.Actions = int(math.Ceil(float64(st.ToXP-st.FromXP) / m.XP))
			st.GP = int(float64(st.Actions) * m.GP)
		}
		// Hours sum each rate band the step crosses.
		for _, sg := range segs {
			lo, hi := max(sg.from, st.FromXP), min(sg.to, st.ToXP)
			if r := m.rateAt(sg.from); hi > lo && r > 0 {
				st.Hours += float64(hi-lo) / r
			}
		}
		p.Actions += st.Actions
		p.GP += st.GP
		p.Hours += st.Hours
	}
	return p
}
