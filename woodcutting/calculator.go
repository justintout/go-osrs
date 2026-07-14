package woodcutting

import (
	"math"
)

type Action struct {
	Tree   Tree
	Count  int
}

func Calculate(startXp, targetXp int, disabledTrees []string, overrideLevels map[string]int) []Action {
	var actions []Action
	currentXp := startXp

	isTreeDisabled := func(treeName string) bool {
		for _, disabledTree := range disabledTrees {
			if treeName == disabledTree {
				return true
			}
		}
		return false
	}

	getTreeLevel := func(tree Tree) int {
		if override, ok := overrideLevels[tree.Name]; ok {
			return override
		}
		return tree.Level
	}

	for currentXp < targetXp {
		currentLevel := LevelForXP(currentXp)
		bestTree := Tree{}
		bestXpRate := 0.0

		for _, tree := range Trees {
			treeLevel := getTreeLevel(tree)
			if currentLevel >= treeLevel && !isTreeDisabled(tree.Name) {
				if tree.XP > bestXpRate {
					bestTree = tree
					bestXpRate = tree.XP
				}
			}
		}

		if bestXpRate == 0 {
			// No more available trees to chop
			break
		}

		// Find the level of the next available tree that unlocks a better
		// XP rate. If none exists, the only remaining breakpoint is the
		// target itself, so chop the best current tree straight to targetXp.
		nextTreeLevel := 0
		for _, tree := range Trees {
			treeLevel := getTreeLevel(tree)
			if treeLevel > currentLevel && !isTreeDisabled(tree.Name) {
				if nextTreeLevel == 0 || treeLevel < nextTreeLevel {
					nextTreeLevel = treeLevel
				}
			}
		}

		xpToNextBreakpoint := targetXp
		if nextTreeLevel != 0 {
			xpForNextTreeLevel := XPForLevel(nextTreeLevel)
			if xpForNextTreeLevel < xpToNextBreakpoint {
				xpToNextBreakpoint = xpForNextTreeLevel
			}
		}
		xpNeeded := xpToNextBreakpoint - currentXp

		if xpNeeded <= 0 {
			// We have reached the next breakpoint, re-evaluate the best tree
			currentXp = xpToNextBreakpoint
			continue
		}

		logsNeeded := int(math.Ceil(float64(xpNeeded) / bestTree.XP))

		actions = append(actions, Action{Tree: bestTree, Count: logsNeeded})
		currentXp += int(float64(logsNeeded) * bestTree.XP)
	}

	return actions
}