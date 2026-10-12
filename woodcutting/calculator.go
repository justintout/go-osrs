// Package woodcutting plans woodcutting training by tree.
package woodcutting

import (
	"slices"

	"github.com/justintout/go-osrs/planner"
)

// Action is a number of logs to cut from one tree.
type Action struct {
	Tree  Tree
	Count int
}

// Calculate plans the logs to cut from startXP to targetXP, using the
// tree with the most XP per log at each level. Disabled trees are skipped,
// and overrideLevels delays when a tree is used.
func Calculate(startXP, targetXP int, disabledTrees []string, overrideLevels map[string]int) ([]Action, error) {
	var methods []planner.Method
	byName := map[string]Tree{}
	for _, t := range Trees {
		if slices.Contains(disabledTrees, t.Name) {
			continue
		}
		level := t.Level
		if o, ok := overrideLevels[t.Name]; ok {
			level = max(o, t.Level)
		}
		methods = append(methods, planner.Method{Name: t.Name, Level: level, XP: t.XP})
		byName[t.Name] = t
	}
	plan, err := planner.Solve(methods, startXP, targetXP, planner.Goal{Objective: planner.Actions})
	if err != nil {
		return nil, err
	}
	actions := make([]Action, len(plan.Steps))
	for i, s := range plan.Steps {
		actions[i] = Action{Tree: byName[s.Method.Name], Count: s.Actions}
	}
	return actions, nil
}

// Methods returns the trees as planner methods.
func Methods() []planner.Method {
	methods := make([]planner.Method, len(Trees))
	for i, t := range Trees {
		methods[i] = planner.Method{Name: t.Name, Level: t.Level, XP: t.XP}
	}
	return methods
}
