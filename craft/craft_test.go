package craft

import (
	"slices"
	"strings"
	"testing"

	"github.com/justintout/go-osrs/wiki"
)

func TestTree(t *testing.T) {
	g := New([]wiki.Recipe{
		{Page: "Rune platebody", Output: wiki.RecipeItem{Name: "Rune platebody", Quantity: 1},
			Materials: []wiki.RecipeItem{{Name: "Runite bar", Quantity: 5}}, Skills: []wiki.RecipeSkill{{Name: "Smithing", Level: 99, XP: 375}}},
		{Page: "Runite bar", Output: wiki.RecipeItem{Name: "Runite bar", Quantity: 1},
			Materials: []wiki.RecipeItem{{Name: "Runite ore", Quantity: 1}, {Name: "Coal", Quantity: 8}}, Skills: []wiki.RecipeSkill{{Name: "Smithing", Level: 85, XP: 50}}},
		// A worse alternative on another page is not picked.
		{Page: "Superheat Item", Output: wiki.RecipeItem{Name: "Runite bar", Quantity: 1},
			Materials: []wiki.RecipeItem{{Name: "Runite ore", Quantity: 1}, {Name: "Nature rune", Quantity: 1}}, Skills: []wiki.RecipeSkill{{Name: "Magic", Level: 43}}},
		// A recipe that consumes its own output is ignored.
		{Page: "Coal", Output: wiki.RecipeItem{Name: "Coal", Quantity: 2}, Materials: []wiki.RecipeItem{{Name: "Coal", Quantity: 1}}},
	})
	tree, err := g.Tree("rune platebody", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []Material{{Item: "Coal", Quantity: 80}, {Item: "Runite ore", Quantity: 10}}
	if got := tree.Raw(); !slices.Equal(got, want) {
		t.Errorf("raw = %+v", got)
	}
	if got := tree.Skills(); len(got) != 1 || got[0] != (SkillTotal{Skill: "Smithing", Level: 99, XP: 2*375 + 10*50}) {
		t.Errorf("skills = %+v", got)
	}
	if bar := tree.Inputs[0]; bar.Alternatives != 1 || bar.Crafts != 10 {
		t.Errorf("runite bar = %+v", bar)
	}
	if !strings.Contains(tree.Text(), "└── Runite bar x10 (85 Smithing)") || !strings.Contains(tree.Mermaid(), `(["Coal x80"])`) {
		t.Errorf("text:\n%s\nmermaid:\n%s", tree.Text(), tree.Mermaid())
	}
	if _, err := g.Tree("Nothing", 1); err == nil {
		t.Error("unknown item: want error")
	}
}
