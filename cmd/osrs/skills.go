package main

import (
	"context"
	"fmt"

	"github.com/justintout/go-osrs/skills"
)

type xpIn struct {
	XP     int `json:"xp,omitempty" jsonschema:"current XP"`
	Level  int `json:"level,omitempty" jsonschema:"current level, used when xp is not set"`
	Target int `json:"target,omitempty" jsonschema:"target level"`
}

type xpOut struct {
	XP           int `json:"xp"`
	Level        int `json:"level"`
	VirtualLevel int `json:"virtual_level"`
	Target       int `json:"target,omitempty"`
	TargetXP     int `json:"target_xp,omitempty"`
	Remaining    int `json:"remaining,omitempty"`
}

func (o xpOut) text() string {
	s := fmt.Sprintf("level %d (virtual %d) at %s xp\n", o.Level, o.VirtualLevel, num(int64(o.XP)))
	if o.Target > 0 {
		s += fmt.Sprintf("level %d needs %s xp: %s to go\n", o.Target, num(int64(o.TargetXP)), num(int64(o.Remaining)))
	}
	return s
}

var xpCmd = newCommand("xp", "convert between XP and levels, and find the XP left to a target level",
	func(_ context.Context, _ *deps, in xpIn) (xpOut, error) {
		xp := in.XP
		if xp == 0 && in.Level > 0 {
			xp = skills.XPForLevel(in.Level)
		}
		if xp < 0 || xp > skills.MaxXP {
			return xpOut{}, fmt.Errorf("xp must be between 0 and %d", skills.MaxXP)
		}
		out := xpOut{XP: xp, Level: skills.Level(xp), VirtualLevel: skills.VirtualLevel(xp)}
		if in.Target != 0 {
			if in.Target < 1 || in.Target > skills.MaxVirtualLevel {
				return xpOut{}, fmt.Errorf("target must be between 1 and %d", skills.MaxVirtualLevel)
			}
			out.Target, out.TargetXP, out.Remaining = in.Target, skills.XPForLevel(in.Target), skills.XPToLevel(xp, in.Target)
		}
		return out, nil
	})

type combatIn struct {
	Attack    int `json:"attack" jsonschema:"Attack level"`
	Strength  int `json:"strength" jsonschema:"Strength level"`
	Defence   int `json:"defence" jsonschema:"Defence level"`
	Hitpoints int `json:"hitpoints" jsonschema:"Hitpoints level"`
	Ranged    int `json:"ranged" jsonschema:"Ranged level"`
	Magic     int `json:"magic" jsonschema:"Magic level"`
	Prayer    int `json:"prayer" jsonschema:"Prayer level"`
}

type combatOut struct {
	CombatLevel int `json:"combat_level"`
}

func (o combatOut) text() string { return fmt.Sprintf("combat level %d\n", o.CombatLevel) }

var combatCmd = newCommand("combat", "calculate combat level from combat skill levels",
	func(_ context.Context, _ *deps, in combatIn) (combatOut, error) {
		levels := []*int{&in.Attack, &in.Strength, &in.Defence, &in.Ranged, &in.Magic, &in.Prayer}
		for _, l := range levels {
			if *l == 0 {
				*l = 1
			}
		}
		if in.Hitpoints == 0 {
			in.Hitpoints = 10
		}
		for _, l := range append(levels, &in.Hitpoints) {
			if *l < 1 || *l > skills.MaxLevel {
				return combatOut{}, fmt.Errorf("levels must be between 1 and %d", skills.MaxLevel)
			}
		}
		return combatOut{skills.CombatLevel(in.Attack, in.Strength, in.Defence, in.Hitpoints, in.Ranged, in.Magic, in.Prayer)}, nil
	})
