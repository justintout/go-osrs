package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Recipe is a way to make an item from other items.
type Recipe struct {
	Page    string `json:"page"`
	Members bool   `json:"members"`
	// Ticks is the game ticks per action, or 0 when the wiki does not say.
	// A tick is 0.6 seconds.
	Ticks     float64       `json:"ticks"`
	Materials []RecipeItem  `json:"materials"`
	Output    RecipeItem    `json:"output"`
	Skills    []RecipeSkill `json:"skills"`
}

// RecipeItem is an item a recipe consumes or makes.
type RecipeItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
}

// RecipeSkill is a skill a recipe needs, and the XP it gives.
type RecipeSkill struct {
	Name  string  `json:"name"`
	Level int     `json:"level"`
	XP    float64 `json:"xp"`
}

// Recipes returns every recipe on the wiki that makes an item, paging
// through the bucket. Wiki recipes without an output item, such as
// activities, are left out. It makes one request per 5000 rows.
func (c *Client) Recipes(ctx context.Context) ([]Recipe, error) {
	rows, err := runPaged[struct {
		Page string `json:"page_name"`
		JSON string `json:"production_json"`
	}](ctx, c, Bucket("recipe").Select("page_name", "production_json"))
	if err != nil {
		return nil, err
	}
	var recipes []Recipe
	for _, r := range rows {
		rec, ok, err := parseRecipe(r.Page, r.JSON)
		if err != nil {
			return nil, err
		}
		if ok {
			recipes = append(recipes, rec)
		}
	}
	return recipes, nil
}

// parseRecipe reports false for recipes with no output item, which the
// wiki stores as an empty string instead of an object.
func parseRecipe(page, raw string) (Recipe, bool, error) {
	var p struct {
		Ticks     flexNum         `json:"ticks"`
		Members   json.RawMessage `json:"members"`
		Materials []struct {
			Name     string  `json:"name"`
			Quantity flexNum `json:"quantity"`
		} `json:"materials"`
		Output json.RawMessage `json:"output"`
		Skills []struct {
			Name  string  `json:"name"`
			Level flexNum `json:"level"`
			XP    flexNum `json:"experience"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Recipe{}, false, fmt.Errorf("wiki: decode recipe %q: %w", page, err)
	}
	if len(p.Output) == 0 || p.Output[0] != '{' {
		return Recipe{}, false, nil
	}
	var out struct {
		Name     string  `json:"name"`
		Quantity flexNum `json:"quantity"`
	}
	if err := json.Unmarshal(p.Output, &out); err != nil {
		return Recipe{}, false, fmt.Errorf("wiki: decode recipe %q output: %w", page, err)
	}
	r := Recipe{
		Page: page, Members: string(p.Members) == "true", Ticks: float64(p.Ticks),
		Output: RecipeItem{Name: out.Name, Quantity: float64(out.Quantity)},
	}
	for _, m := range p.Materials {
		r.Materials = append(r.Materials, RecipeItem{Name: m.Name, Quantity: float64(m.Quantity)})
	}
	for _, s := range p.Skills {
		r.Skills = append(r.Skills, RecipeSkill{Name: s.Name, Level: int(s.Level), XP: float64(s.XP)})
	}
	return r, true, nil
}

// flexNum decodes a number the wiki sends as a JSON number or a string.
// Text that is not a number, such as "varies", decodes as 0.
type flexNum float64

func (f *flexNum) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		n = 0
	}
	*f = flexNum(n)
	return nil
}
