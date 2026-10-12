package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/justintout/go-osrs/examples/saplings/templates"
	"github.com/justintout/go-osrs/prices"
)

const iconBaseURL = "https://oldschool.runescape.wiki/images/"

var treePairs = map[string]string{
	"Acorn":                 "Oak sapling",
	"Willow seed":           "Willow sapling",
	"Maple seed":            "Maple sapling",
	"Yew seed":              "Yew sapling",
	"Magic seed":            "Magic sapling",
	"Apple tree seed":       "Apple sapling",
	"Banana tree seed":      "Banana sapling",
	"Orange tree seed":      "Orange sapling",
	"Curry tree seed":       "Curry sapling",
	"Pineapple seed":        "Pineapple sapling",
	"Papaya tree seed":      "Papaya sapling",
	"Palm tree seed":        "Palm sapling",
	"Calquat tree seed":     "Calquat sapling",
	"Dragonfruit tree seed": "Dragonfruit sapling",
	"Celastrus seed":        "Celastrus sapling",
}

// geTax is the Grand Exchange sale tax: 2%, rounded down, capped at 5m.
// Rounding down exempts items under 50gp.
func geTax(price int) int {
	return min(price*2/100, 5_000_000)
}

func comma(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, r := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func main() {
	c := prices.NewClient("go-osrs/0.0.1 (https://github.com/justintout/go-osrs/cmd/saplings)")

	mapping, err := c.Mapping(context.Background())
	if err != nil {
		log.Fatalf("fetch item mapping: %s", err)
	}
	itemMap := make(map[string]prices.Item, len(mapping))
	for _, i := range mapping {
		itemMap[i.Name] = i
	}
	for seed, sapling := range treePairs {
		for _, name := range []string{seed, sapling} {
			if _, ok := itemMap[name]; !ok {
				log.Fatalf("item %q not in wiki mapping", name)
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		prices, err := c.Latest(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("fetch prices: %s", err), http.StatusBadGateway)
			return
		}

		var results []templates.ProfitResult
		for seedName, saplingName := range treePairs {
			seed, sapling := itemMap[seedName], itemMap[saplingName]
			seedPrice, saplingPrice := prices[seed.ID].Low, prices[sapling.ID].High
			results = append(results, templates.ProfitResult{
				Name:             strings.TrimSuffix(seedName, " seed"),
				Profit:           saplingPrice - geTax(saplingPrice) - seedPrice,
				SeedBuyPrice:     seedPrice,
				SaplingSellPrice: saplingPrice,
				SeedImage:        iconBaseURL + strings.ReplaceAll(seed.Icon, " ", "_"),
				SaplingImage:     iconBaseURL + strings.ReplaceAll(sapling.Icon, " ", "_"),
				BuyLimitCost:     comma(seedPrice * seed.Limit),
			})
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Profit > results[j].Profit
		})

		templates.Index(results).Render(r.Context(), w)
	})

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
