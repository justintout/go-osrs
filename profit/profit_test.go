package profit

import (
	"testing"

	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

func TestAlchsAndRecipes(t *testing.T) {
	items := []prices.Item{
		{ID: 1, Name: "Staff", HighAlch: 1000, Limit: 100},
		{ID: 2, Name: "Junk", HighAlch: 10, Limit: 100},
		{ID: 3, Name: "Thin", HighAlch: 5000, Limit: 100},
		{ID: 4, Name: "Bar", Limit: 1000},
		{ID: 5, Name: "Plate"},
		{ID: NatureRuneID, Name: "Nature rune"},
	}
	latest := prices.Prices{
		1: {High: 800}, 2: {High: 50}, 3: {High: 100}, NatureRuneID: {High: 100},
		4: {High: 1000}, 5: {Low: 5000},
	}
	vols := map[int]int{1: 1000, 2: 1000, 3: 1, 4: 1000, 5: 1000}

	alchs := Alchs(items, latest, vols, 500)
	if len(alchs) != 1 || alchs[0].Name != "Staff" || alchs[0].Profit != 100 || alchs[0].ProfitPerLimit != 10_000 {
		t.Errorf("alchs = %+v", alchs)
	}

	recipes := []wiki.Recipe{
		{Page: "Plate", Materials: []wiki.RecipeItem{{Name: "Bar", Quantity: 3}}, Output: wiki.RecipeItem{Name: "Plate", Quantity: 1}, Ticks: 5},
		{Page: "Loss", Materials: []wiki.RecipeItem{{Name: "Bar", Quantity: 9}}, Output: wiki.RecipeItem{Name: "Plate", Quantity: 1}},
		{Page: "Unknown", Materials: []wiki.RecipeItem{{Name: "Mystery", Quantity: 1}}, Output: wiki.RecipeItem{Name: "Plate", Quantity: 1}},
	}
	got := Recipes(recipes, prices.NewCatalog(items), latest, vols, 500)
	// 5000 sale less 100 tax, less 3 bars at 1000.
	if len(got) != 1 || got[0].Profit != 1900 || got[0].ProfitPerHour != 1900*1200 {
		t.Errorf("recipes = %+v", got)
	}
}
