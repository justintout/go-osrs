package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/justintout/osrs/cmd/saplings/templates"
	"github.com/justintout/osrs/items"
)

var (
	itemMap     map[string]int
	mappingOnce sync.Once
	treePairs   map[string]string
)

func main() {
	treePairs = map[string]string{
		"Acorn":               "Oak sapling",
		"Willow seed":         "Willow sapling",
		"Maple seed":          "Maple sapling",
		"Yew seed":            "Yew sapling",
		"Magic seed":          "Magic sapling",
		"Apple tree seed":     "Apple sapling",
		"Banana tree seed":    "Banana sapling",
		"Orange tree seed":    "Orange sapling",
		"Curry tree seed":     "Curry sapling",
		"Pineapple tree seed": "Pineapple sapling",
		"Papaya tree seed":    "Papaya sapling",
		"Palm tree seed":      "Palm sapling",
		"Calquat tree seed":   "Calquat sapling",
	}

	c := items.NewClient("go-osrs/0.0.1 (https://github.com/justintout/go-osrs/cmd/saplings)")
	mappingOnce.Do(func() {
		itemMapping, err := c.Mapping()
		if err != nil {
			panic(err)
		}
		itemMap = make(map[string]int)
		for _, i := range itemMapping {
			itemMap[i.Name] = i.ID
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		templates.Index().Render(r.Context(), w)
	})

	http.HandleFunc("/profits", func(w http.ResponseWriter, r *http.Request) {
		var results []templates.ProfitResult

		for seedName, saplingName := range treePairs {
			seedID, ok := itemMap[seedName]
			if !ok {
				results = append(results, templates.ProfitResult{Name: strings.TrimSuffix(seedName, " seed"), Profit: "Seed not found"})
				continue
			}
			saplingID, ok := itemMap[saplingName]
			if !ok {
				results = append(results, templates.ProfitResult{Name: strings.TrimSuffix(seedName, " seed"), Profit: "Sapling not found"})
				continue
			}

			seedPrice, err := c.LatestFor(seedID)
			if err != nil {
				results = append(results, templates.ProfitResult{Name: strings.TrimSuffix(seedName, " seed"), Profit: "Error fetching seed price"})
				continue
			}

			saplingPrice, err := c.LatestFor(saplingID)
			if err != nil {
				results = append(results, templates.ProfitResult{Name: strings.TrimSuffix(seedName, " seed"), Profit: "Error fetching sapling price"})
				continue
			}

			profit := saplingPrice.Low - seedPrice.High
			results = append(results, templates.ProfitResult{Name: strings.TrimSuffix(seedName, " seed"), Profit: fmt.Sprintf("%d gp", profit)})
		}

		templates.Profits(results).Render(r.Context(), w)
	})

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
