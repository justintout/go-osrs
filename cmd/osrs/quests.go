package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/quests"
)

type questIn struct {
	Quest  string `json:"quest" arg:"" jsonschema:"quest name, such as Song of the Elves"`
	Player string `json:"player,omitempty" jsonschema:"compare against this player's levels"`
}

type questSkill struct {
	Skill     string `json:"skill"`
	Level     int    `json:"level"`
	Boostable bool   `json:"boostable"`
	Have      int    `json:"have,omitempty"`
	Met       *bool  `json:"met,omitempty"`
}

type questOut struct {
	Quest         string       `json:"quest"`
	Difficulty    string       `json:"difficulty"`
	Length        string       `json:"length"`
	Skills        []questSkill `json:"skills"`
	QuestPoints   int          `json:"quest_points,omitempty"`
	CombatLevel   int          `json:"combat_level,omitempty"`
	Prerequisites []string     `json:"prerequisites"`
	Player        string       `json:"player,omitempty"`
	MissingSkills int          `json:"missing_skills,omitempty"`
}

func (o questOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s, %s)\n", o.Quest, o.Difficulty, o.Length)
	if o.QuestPoints > 0 {
		fmt.Fprintf(&b, "quest points %d\n", o.QuestPoints)
	}
	if o.CombatLevel > 0 {
		fmt.Fprintf(&b, "combat level %d\n", o.CombatLevel)
	}
	if len(o.Skills) > 0 {
		b.WriteString("\nskills, across all prerequisites:\n")
		headers := []string{"SKILL", "LEVEL", "BOOSTABLE"}
		if o.Player != "" {
			headers = append(headers, "HAVE", "")
		}
		t := newTable(&b, headers...)
		for _, s := range o.Skills {
			row := []string{s.Skill, strconv.Itoa(s.Level), yesNo(s.Boostable)}
			if s.Met != nil {
				mark := "ok"
				if !*s.Met {
					mark = fmt.Sprintf("need %d more", s.Level-s.Have)
				}
				row = append(row, strconv.Itoa(s.Have), mark)
			}
			t.row(row...)
		}
		t.flush()
	}
	if len(o.Prerequisites) > 0 {
		fmt.Fprintf(&b, "\nprerequisite quests, in completion order:\n  %s\n", strings.Join(o.Prerequisites, "\n  "))
	}
	if o.Player != "" {
		if o.MissingSkills == 0 {
			fmt.Fprintf(&b, "\n%s has every skill level needed. Quest completion is not public, so prerequisites are not checked.\n", o.Player)
		} else {
			fmt.Fprintf(&b, "\n%s is short on %d skills. Skills too low to rank on the hiscores count as level 1.\n", o.Player, o.MissingSkills)
		}
	}
	return b.String()
}

var questCmd = newCommand("quest", "get a quest's full requirements across its prerequisite quests, optionally checked against a player's levels",
	func(ctx context.Context, d *deps, in questIn) (questOut, error) {
		if err := need("quest", in.Quest); err != nil {
			return questOut{}, err
		}
		g, err := questGraph(ctx, d)
		if err != nil {
			return questOut{}, err
		}
		r, err := g.Requirements(in.Quest)
		if err != nil {
			return questOut{}, err
		}
		out := questOut{Quest: r.Quest.Name, Difficulty: r.Quest.Difficulty, Length: r.Quest.Length, QuestPoints: r.QuestPoints, CombatLevel: r.CombatLevel, Prerequisites: r.Prerequisites}
		var levels map[string]int
		if in.Player != "" {
			if levels, err = playerLevels(ctx, d, in.Player); err != nil {
				return questOut{}, err
			}
			out.Player = in.Player
		}
		for _, s := range r.Skills {
			qs := questSkill{Skill: s.Skill, Level: s.Level, Boostable: s.Boostable}
			if levels != nil {
				have := levels[strings.ToLower(s.Skill)]
				met := have >= s.Level
				qs.Have, qs.Met = have, &met
				if !met {
					out.MissingSkills++
				}
			}
			out.Skills = append(out.Skills, qs)
		}
		return out, nil
	})

type questsIn struct {
	Player string `json:"player" arg:"" jsonschema:"player name"`
}

type blockedQuest struct {
	Quest   string   `json:"quest"`
	Missing []string `json:"missing"`
	// Levels is the total levels still needed across missing skills.
	Levels int `json:"levels"`
}

type questsOut struct {
	Player string `json:"player"`
	// Ready lists quests whose full-tree skill requirements the player
	// meets. Completion is not public, so finished quests are included.
	Ready   []string       `json:"ready"`
	Blocked []blockedQuest `json:"blocked"`
}

func (o questsOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s meets the skill requirements for %d quests and is short for %d.\n", o.Player, len(o.Ready), len(o.Blocked))
	b.WriteString("Quest completion is not public, so finished quests are included.\nSkills too low to rank on the hiscores count as level 1.\n\nclosest blocked quests:\n")
	t := newTable(&b, "QUEST", "LEVELS SHORT", "MISSING")
	for _, q := range top(o.Blocked, 15) {
		t.row(q.Quest, strconv.Itoa(q.Levels), strings.Join(q.Missing, ", "))
	}
	t.flush()
	return b.String()
}

var questsCmd = newCommand("quests", "sort every quest into those a player has the skill levels for and those they are short on, closest first",
	func(ctx context.Context, d *deps, in questsIn) (questsOut, error) {
		if err := need("player", in.Player); err != nil {
			return questsOut{}, err
		}
		levels, err := playerLevels(ctx, d, in.Player)
		if err != nil {
			return questsOut{}, err
		}
		g, err := questGraph(ctx, d)
		if err != nil {
			return questsOut{}, err
		}
		out := questsOut{Player: in.Player}
		for _, name := range g.Names() {
			r, err := g.Requirements(name)
			if err != nil {
				return questsOut{}, err
			}
			unmet := r.Unmet(levels)
			if len(unmet) == 0 {
				out.Ready = append(out.Ready, name)
				continue
			}
			bq := blockedQuest{Quest: name}
			for _, u := range unmet {
				bq.Missing = append(bq.Missing, fmt.Sprintf("%s %d/%d", u.Skill, u.Have, u.Level))
				bq.Levels += u.Level - u.Have
			}
			out.Blocked = append(out.Blocked, bq)
		}
		sort.SliceStable(out.Blocked, func(i, j int) bool { return out.Blocked[i].Levels < out.Blocked[j].Levels })
		return out, nil
	})

func questGraph(ctx context.Context, d *deps) (*quests.Graph, error) {
	qs, err := d.wiki.Quests(ctx)
	if err != nil {
		return nil, err
	}
	return quests.New(qs), nil
}

// playerLevels maps lowercase skill names to a player's levels.
func playerLevels(ctx context.Context, d *deps, player string) (map[string]int, error) {
	p, err := d.hiscores.Lookup(ctx, player, hiscores.Regular)
	if errors.Is(err, osrs.ErrNotFound) {
		return nil, fmt.Errorf("%s is not on the hiscores", player)
	}
	if err != nil {
		return nil, err
	}
	levels := map[string]int{}
	for _, s := range p.Skills {
		levels[strings.ToLower(s.Name)] = max(s.Level, 1)
	}
	return levels, nil
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
