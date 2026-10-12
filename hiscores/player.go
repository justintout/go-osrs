package hiscores

import (
	"context"
	"net/url"
	"strings"

	"github.com/justintout/go-osrs/skills"
)

// Player is a player's entry on one mode's hiscores.
type Player struct {
	Name       string     `json:"name"`
	Skills     []Skill    `json:"skills"`
	Activities []Activity `json:"activities"`
}

// Skill is a player's standing in a skill. Rank and XP are -1 when the
// player is unranked.
type Skill struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rank  int    `json:"rank"`
	Level int    `json:"level"`
	XP    int64  `json:"xp"`
}

// Activity is a player's standing in a boss, minigame, or clue tier. Rank
// and Score are -1 when the player is unranked.
type Activity struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rank  int    `json:"rank"`
	Score int    `json:"score"`
}

// Lookup returns a player's entry on the given mode's hiscores. It returns
// an error matching osrs.ErrNotFound when the player is not on that board.
//
// Skill and activity names come from Jagex, so new content appears without
// a library update.
func (c *Client) Lookup(ctx context.Context, player string, mode Mode) (*Player, error) {
	u, err := c.modeURL(mode, "index_lite.json", url.Values{"player": {player}})
	if err != nil {
		return nil, err
	}
	var p Player
	if err := c.t.GetJSON(ctx, u, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Skill returns the named skill, ignoring case.
func (p *Player) Skill(name string) (Skill, bool) {
	for _, s := range p.Skills {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}
	return Skill{}, false
}

// Activity returns the named activity, ignoring case.
func (p *Player) Activity(name string) (Activity, bool) {
	for _, a := range p.Activities {
		if strings.EqualFold(a.Name, name) {
			return a, true
		}
	}
	return Activity{}, false
}

// Overall returns the player's Overall skill entry.
func (p *Player) Overall() Skill {
	s, _ := p.Skill("Overall")
	return s
}

// CombatLevel returns the player's combat level from their combat skills.
// Unranked skills count at their minimum level.
func (p *Player) CombatLevel() int {
	level := func(s skills.Skill) int {
		sk, ok := p.Skill(string(s))
		if !ok || sk.Level < 1 {
			if s == skills.Hitpoints {
				return 10
			}
			return 1
		}
		return sk.Level
	}
	return skills.CombatLevel(
		level(skills.Attack), level(skills.Strength), level(skills.Defence), level(skills.Hitpoints),
		level(skills.Ranged), level(skills.Magic), level(skills.Prayer),
	)
}
