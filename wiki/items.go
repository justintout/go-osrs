package wiki

import (
	"context"
	"strconv"
)

// Item is one version of an item's infobox. Pages like "Games necklace"
// have a version per charge count.
type Item struct {
	Page        string   `json:"page_name"`
	Version     string   `json:"version_anchor"`
	Name        string   `json:"item_name"`
	RawIDs      []string `json:"item_id"`
	Members     Flag     `json:"is_members_only"`
	Tradeable   Flag     `json:"tradeable"`
	Examine     string   `json:"examine"`
	Value       int      `json:"value"`
	HighAlch    int      `json:"high_alchemy_value"`
	Weight      float64  `json:"weight"`
	BuyLimit    int      `json:"buy_limit"`
	ReleaseDate string   `json:"release_date"`
	Quest       string   `json:"quest"`
}

// IDs returns the item's numeric IDs. Entries the wiki marks as
// non-numeric, such as beta IDs, are skipped.
func (i Item) IDs() []int {
	var ids []int
	for _, s := range i.RawIDs {
		if id, err := strconv.Atoi(s); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// Item returns every version of the item on the given wiki page.
func (c *Client) Item(ctx context.Context, page string) ([]Item, error) {
	var items []Item
	q := Bucket("infobox_item").
		Select("page_name", "version_anchor", "item_name", "item_id", "is_members_only", "tradeable", "examine",
			"value", "high_alchemy_value", "weight", "buy_limit", "release_date", "quest").
		Where("page_name", page)
	return items, c.Run(ctx, q, &items)
}

// Bonuses are an equipment item's combat stats.
type Bonuses struct {
	Page          string  `json:"page_name"`
	Slot          string  `json:"equipment_slot"`
	StabAttack    int     `json:"stab_attack_bonus"`
	SlashAttack   int     `json:"slash_attack_bonus"`
	CrushAttack   int     `json:"crush_attack_bonus"`
	MagicAttack   int     `json:"magic_attack_bonus"`
	RangedAttack  int     `json:"range_attack_bonus"`
	StabDefence   int     `json:"stab_defence_bonus"`
	SlashDefence  int     `json:"slash_defence_bonus"`
	CrushDefence  int     `json:"crush_defence_bonus"`
	MagicDefence  int     `json:"magic_defence_bonus"`
	RangedDefence int     `json:"range_defence_bonus"`
	Strength      int     `json:"strength_bonus"`
	RangedStr     int     `json:"ranged_strength_bonus"`
	MagicDamage   float64 `json:"magic_damage_bonus"`
	Prayer        int     `json:"prayer_bonus"`
	AttackSpeed   int     `json:"weapon_attack_speed"`
	AttackRange   string  `json:"weapon_attack_range"`
	CombatStyle   string  `json:"combat_style"`
}

// Bonuses returns the equipment stats on an item's wiki page, one entry
// per infobox. The bonuses bucket does not record version names.
func (c *Client) Bonuses(ctx context.Context, page string) ([]Bonuses, error) {
	var b []Bonuses
	q := Bucket("infobox_bonuses").
		Select("page_name", "equipment_slot", "stab_attack_bonus", "slash_attack_bonus",
			"crush_attack_bonus", "magic_attack_bonus", "range_attack_bonus", "stab_defence_bonus",
			"slash_defence_bonus", "crush_defence_bonus", "magic_defence_bonus", "range_defence_bonus",
			"strength_bonus", "ranged_strength_bonus", "magic_damage_bonus", "prayer_bonus",
			"weapon_attack_speed", "weapon_attack_range", "combat_style").
		Where("page_name", page)
	return b, c.Run(ctx, q, &b)
}
