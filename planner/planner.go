// Package planner plans skill training: which method to use from each
// level to a target, switching when a better method unlocks.
package planner

import (
	"errors"
	"fmt"
	"math"

	"github.com/justintout/go-osrs/skills"
)

// Method is one way to train a skill.
type Method struct {
	Name  string  `json:"name"`
	Level int     `json:"level"`
	XP    float64 `json:"xp"`
	// GP is the coins gained per action. A negative GP is a cost.
	GP float64 `json:"gp"`
	// MaxActions caps how many times the method can be used, such as by
	// how much of its output the market can absorb. 0 means no cap.
	MaxActions int `json:"max_actions,omitempty"`
}

// Goal ranks methods.
type Goal int

const (
	// FewestActions picks the most XP per action.
	FewestActions Goal = iota
	// BestGP picks the most GP per XP, which is the cheapest method when
	// every method costs money.
	BestGP
)

func (g Goal) score(m Method) float64 {
	if g == BestGP {
		return m.GP / m.XP
	}
	return m.XP
}

// Step is a stretch of training with one method.
type Step struct {
	Method  Method `json:"method"`
	FromXP  int    `json:"from_xp"`
	ToXP    int    `json:"to_xp"`
	Actions int    `json:"actions"`
	// GP is the coins gained over the step. A negative GP is a cost.
	GP int `json:"gp"`
}

// Plan returns training steps from fromXP to toXP. At each level it uses
// the best available method for goal, and it switches when a method that
// scores higher unlocks or the current method reaches its MaxActions.
// Methods with no XP are ignored.
func Plan(methods []Method, fromXP, toXP int, goal Goal) ([]Step, error) {
	if fromXP < 0 || toXP > skills.MaxXP {
		return nil, fmt.Errorf("planner: xp range %d to %d is outside 0 to %d", fromXP, toXP, skills.MaxXP)
	}
	var usable []Method
	for _, m := range methods {
		if m.XP > 0 {
			usable = append(usable, m)
		}
	}
	if len(usable) == 0 {
		return nil, errors.New("planner: no methods give XP")
	}

	var steps []Step
	used := make([]int, len(usable))
	xp := float64(fromXP)
	for xp < float64(toXP) {
		level := skills.Level(int(xp))
		i, ok := bestAt(usable, used, level, goal)
		if !ok {
			return nil, fmt.Errorf("planner: no method available at level %d", level)
		}
		best := usable[i]
		end := float64(toXP)
		if next, ok := nextBetter(usable, level, goal.score(best), goal); ok {
			end = math.Min(end, float64(skills.XPForLevel(next)))
		}
		actions := int(math.Ceil((end - xp) / best.XP))
		if best.MaxActions > 0 {
			actions = min(actions, best.MaxActions-used[i])
		}
		used[i] += actions
		gained := float64(actions) * best.XP
		if n := len(steps); n > 0 && steps[n-1].Method == best {
			steps[n-1].Actions += actions
			steps[n-1].ToXP = int(xp + gained)
			steps[n-1].GP = int(float64(steps[n-1].Actions) * best.GP)
		} else {
			steps = append(steps, Step{Method: best, FromXP: int(xp), ToXP: int(xp + gained), Actions: actions, GP: int(float64(actions) * best.GP)})
		}
		xp += gained
	}
	return steps, nil
}

// bestAt returns the index of the best method usable at level that has
// not reached its cap.
func bestAt(methods []Method, used []int, level int, goal Goal) (int, bool) {
	best := -1
	for i, m := range methods {
		capped := m.MaxActions > 0 && used[i] >= m.MaxActions
		if m.Level <= level && !capped && (best < 0 || goal.score(m) > goal.score(methods[best])) {
			best = i
		}
	}
	return best, best >= 0
}

// nextBetter returns the lowest level above level that unlocks a method
// scoring higher than score.
func nextBetter(methods []Method, level int, score float64, goal Goal) (int, bool) {
	next, found := 0, false
	for _, m := range methods {
		if m.Level > level && goal.score(m) > score && (!found || m.Level < next) {
			next, found = m.Level, true
		}
	}
	return next, found
}
