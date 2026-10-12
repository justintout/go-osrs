package wiki

import "context"

// Monster is one version of a monster's infobox. Pages like "Vorkath"
// have a version per encounter.
type Monster struct {
	Page           string   `json:"page_name"`
	Version        string   `json:"version_anchor"`
	Name           string   `json:"name"`
	Members        Flag     `json:"is_members_only"`
	Examine        string   `json:"examine"`
	CombatLevel    int      `json:"combat_level"`
	Hitpoints      int      `json:"hitpoints"`
	MaxHit         []string `json:"max_hit"`
	Attributes     []string `json:"attribute"`
	Aggressive     string   `json:"is_aggressive"`
	SlayerLevel    int      `json:"slayer_level"`
	SlayerXP       float64  `json:"slayer_experience"`
	SlayerCategory []string `json:"slayer_category"`
	AssignedBy     []string `json:"assigned_by"`
	AttackLevel    int      `json:"attack_level"`
	StrengthLevel  int      `json:"strength_level"`
	DefenceLevel   int      `json:"defence_level"`
	RangedLevel    int      `json:"ranged_level"`
	MagicLevel     int      `json:"magic_level"`
	StabDefence    int      `json:"stab_defence_bonus"`
	SlashDefence   int      `json:"slash_defence_bonus"`
	CrushDefence   int      `json:"crush_defence_bonus"`
	MagicDefence   int      `json:"magic_defence_bonus"`
	RangedDefence  int      `json:"range_defence_bonus"`
}

// Monster returns every version of the monster on the given wiki page.
func (c *Client) Monster(ctx context.Context, page string) ([]Monster, error) {
	var m []Monster
	q := Bucket("infobox_monster").
		Select("page_name", "version_anchor", "name", "is_members_only", "examine", "combat_level", "hitpoints",
			"max_hit", "attribute", "is_aggressive", "slayer_level", "slayer_experience", "slayer_category",
			"assigned_by", "attack_level", "strength_level", "defence_level", "ranged_level", "magic_level",
			"stab_defence_bonus", "slash_defence_bonus", "crush_defence_bonus", "magic_defence_bonus",
			"range_defence_bonus").
		Where("page_name", page)
	return m, c.Run(ctx, q, &m)
}
