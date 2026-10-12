// Package skills holds skill names and the XP and combat level formulas.
package skills

import "math"

// Skill is a skill name as Jagex writes it on the hiscores.
type Skill string

const (
	Attack       Skill = "Attack"
	Defence      Skill = "Defence"
	Strength     Skill = "Strength"
	Hitpoints    Skill = "Hitpoints"
	Ranged       Skill = "Ranged"
	Prayer       Skill = "Prayer"
	Magic        Skill = "Magic"
	Cooking      Skill = "Cooking"
	Woodcutting  Skill = "Woodcutting"
	Fletching    Skill = "Fletching"
	Fishing      Skill = "Fishing"
	Firemaking   Skill = "Firemaking"
	Crafting     Skill = "Crafting"
	Smithing     Skill = "Smithing"
	Mining       Skill = "Mining"
	Herblore     Skill = "Herblore"
	Agility      Skill = "Agility"
	Thieving     Skill = "Thieving"
	Slayer       Skill = "Slayer"
	Farming      Skill = "Farming"
	Runecraft    Skill = "Runecraft"
	Hunter       Skill = "Hunter"
	Construction Skill = "Construction"
	Sailing      Skill = "Sailing"
)

// All lists every skill in hiscores order.
var All = []Skill{
	Attack, Defence, Strength, Hitpoints, Ranged, Prayer, Magic, Cooking,
	Woodcutting, Fletching, Fishing, Firemaking, Crafting, Smithing, Mining,
	Herblore, Agility, Thieving, Slayer, Farming, Runecraft, Hunter,
	Construction, Sailing,
}

const (
	MaxLevel        = 99
	MaxVirtualLevel = 126
	MaxXP           = 200_000_000
)

// xpTable[i] is the XP needed for level i+1.
var xpTable = func() [MaxVirtualLevel]int {
	var t [MaxVirtualLevel]int
	points := 0
	for l := 1; l < MaxVirtualLevel; l++ {
		points += int(math.Floor(float64(l) + 300*math.Pow(2, float64(l)/7)))
		t[l] = points / 4
	}
	return t
}()

// XPForLevel returns the XP needed to reach level. Levels outside 1..126
// are clamped to that range.
func XPForLevel(level int) int {
	return xpTable[min(max(level, 1), MaxVirtualLevel)-1]
}

// VirtualLevel returns the level for xp, counting past 99 up to 126.
func VirtualLevel(xp int) int {
	for l := MaxVirtualLevel; l > 1; l-- {
		if xp >= xpTable[l-1] {
			return l
		}
	}
	return 1
}

// Level returns the level for xp, capped at 99.
func Level(xp int) int {
	return min(VirtualLevel(xp), MaxLevel)
}

// XPToLevel returns the XP still needed to go from xp to target, or 0 if
// xp already reaches it.
func XPToLevel(xp, target int) int {
	return max(XPForLevel(target)-xp, 0)
}

// CombatLevel returns the combat level for the given skill levels.
func CombatLevel(attack, strength, defence, hitpoints, ranged, magic, prayer int) int {
	// The formula is 1/4*(def+hp+prayer/2) + 13/40*max(att+str, 3/2*ranged,
	// 3/2*magic). Scaling by 40 keeps it in exact integer math.
	base := 10 * (defence + hitpoints + prayer/2)
	best := 13 * max(attack+strength, ranged*3/2, magic*3/2)
	return (base + best) / 40
}
