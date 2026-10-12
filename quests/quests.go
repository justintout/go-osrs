// Package quests resolves a quest's full requirements across its tree of
// prerequisite quests.
package quests

import (
	"fmt"
	"sort"
	"strings"

	"github.com/justintout/go-osrs/wiki"
)

// Graph indexes quests by name.
type Graph struct {
	byName map[string]wiki.Quest
}

// New builds a Graph from wiki.Quests.
func New(quests []wiki.Quest) *Graph {
	g := &Graph{byName: make(map[string]wiki.Quest, len(quests))}
	for _, q := range quests {
		g.byName[strings.ToLower(q.Name)] = q
	}
	return g
}

// Requirements is everything needed to complete a quest, including its
// prerequisites.
type Requirements struct {
	Quest wiki.Quest `json:"quest"`
	// Skills holds the highest level each skill needs anywhere in the
	// tree. Boostable is false if any quest needing that level forbids
	// boosts.
	Skills      []wiki.SkillRequirement `json:"skills"`
	QuestPoints int                     `json:"quest_points,omitempty"`
	CombatLevel int                     `json:"combat_level,omitempty"`
	// Prerequisites lists every prerequisite quest in an order that
	// completes each one after its own prerequisites.
	Prerequisites []string `json:"prerequisites"`
}

// Requirements resolves a quest's full requirements, ignoring case in
// the name.
func (g *Graph) Requirements(name string) (Requirements, error) {
	q, ok := g.byName[strings.ToLower(name)]
	if !ok {
		return Requirements{}, fmt.Errorf("quests: no quest named %q", name)
	}
	r := Requirements{Quest: q}
	skills := map[string]wiki.SkillRequirement{}
	visited := map[string]bool{}
	var visit func(q wiki.Quest)
	visit = func(q wiki.Quest) {
		visited[q.Name] = true
		for _, p := range q.Prerequisites {
			if pq, ok := g.byName[strings.ToLower(p)]; ok && !visited[pq.Name] {
				visit(pq)
				r.Prerequisites = append(r.Prerequisites, pq.Name)
			}
		}
		for _, s := range q.Skills {
			cur, ok := skills[s.Skill]
			switch {
			case !ok || s.Level > cur.Level:
				skills[s.Skill] = s
			case s.Level == cur.Level && !s.Boostable:
				cur.Boostable = false
				skills[s.Skill] = cur
			}
		}
		r.QuestPoints = max(r.QuestPoints, q.QuestPoints)
		r.CombatLevel = max(r.CombatLevel, q.CombatLevel)
	}
	visit(q)
	for _, s := range skills {
		r.Skills = append(r.Skills, s)
	}
	sort.Slice(r.Skills, func(i, j int) bool { return r.Skills[i].Skill < r.Skills[j].Skill })
	return r, nil
}

// Shortfall is how far a player is below a skill requirement.
type Shortfall struct {
	wiki.SkillRequirement
	Have int `json:"have"`
}

// Unmet returns the skill requirements a player's levels do not meet.
// levels maps skill names, ignoring case, to levels.
func (r Requirements) Unmet(levels map[string]int) []Shortfall {
	lower := make(map[string]int, len(levels))
	for k, v := range levels {
		lower[strings.ToLower(k)] = v
	}
	var out []Shortfall
	for _, s := range r.Skills {
		if have := lower[strings.ToLower(s.Skill)]; have < s.Level {
			out = append(out, Shortfall{SkillRequirement: s, Have: have})
		}
	}
	return out
}

// Names returns every quest name, sorted.
func (g *Graph) Names() []string {
	names := make([]string, 0, len(g.byName))
	for _, q := range g.byName {
		names = append(names, q.Name)
	}
	sort.Strings(names)
	return names
}
