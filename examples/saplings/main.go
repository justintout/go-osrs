// Command saplings serves a page comparing tree seed prices with the
// saplings they grow into, to find the most profitable saplings to grow
// and sell.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/justintout/go-osrs/cache"
	"github.com/justintout/go-osrs/examples/saplings/templates"
	"github.com/justintout/go-osrs/prices"
)

const iconBaseURL = "https://oldschool.runescape.wiki/images/"

// trees maps each seed to the sapling it grows into.
var trees = map[string]string{
	"Acorn":                 "Oak sapling",
	"Willow seed":           "Willow sapling",
	"Teak seed":             "Teak sapling",
	"Maple seed":            "Maple sapling",
	"Mahogany seed":         "Mahogany sapling",
	"Yew seed":              "Yew sapling",
	"Magic seed":            "Magic sapling",
	"Redwood tree seed":     "Redwood sapling",
	"Camphor seed":          "Camphor sapling",
	"Ironwood seed":         "Ironwood sapling",
	"Rosewood seed":         "Rosewood sapling",
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

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()
	ua := os.Getenv("OSRS_USER_AGENT")
	if ua == "" {
		ua = "go-osrs saplings example (https://github.com/justintout/go-osrs)"
	}
	c, err := cache.Default()
	if err != nil {
		log.Fatal(err)
	}
	client := prices.NewClient(ua, prices.WithCache(c))

	// Fail at startup on a misspelled name rather than showing a short
	// table later.
	mapping, err := client.Mapping(context.Background())
	if err != nil {
		log.Fatalf("fetch item mapping: %s", err)
	}
	catalog := prices.NewCatalog(mapping)
	for seed, sapling := range trees {
		for _, name := range []string{seed, sapling} {
			if _, ok := catalog.ByName(name); !ok {
				log.Fatalf("item %q is not in the wiki item mapping", name)
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := profits(r.Context(), client, catalog)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := templates.Index(rows).Render(r.Context(), w); err != nil {
			log.Println("render:", err)
		}
	})

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// profits prices every seed and sapling. It buys seeds at the
// instant-sell price and sells saplings at the instant-buy price, which is
// what patient offers on both sides get. The disk cache keeps the price
// API's own freshness, so page reloads do not refetch within a minute.
func profits(ctx context.Context, client *prices.Client, catalog *prices.Catalog) ([]templates.Row, error) {
	latest, err := client.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch prices: %w", err)
	}
	volumes, _, err := client.Volumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch volumes: %w", err)
	}
	var rows []templates.Row
	for seedName, saplingName := range trees {
		seed, _ := catalog.ByName(seedName)
		sapling, _ := catalog.ByName(saplingName)
		buy, sell := latest[seed.ID].Low, latest[sapling.ID].High
		row := templates.Row{
			Tree:          strings.TrimSuffix(strings.TrimSuffix(seedName, " seed"), " tree"),
			SeedPrice:     buy,
			SaplingPrice:  sell,
			SeedImage:     iconBaseURL + strings.ReplaceAll(seed.Icon, " ", "_"),
			SaplingImage:  iconBaseURL + strings.ReplaceAll(sapling.Icon, " ", "_"),
			SeedLimit:     seed.Limit,
			SaplingVolume: volumes[sapling.ID],
			Priced:        buy > 0 && sell > 0,
		}
		if row.Priced {
			row.Profit = prices.AfterTax(sapling.ID, sell) - buy
			row.ProfitPerLimit = row.Profit * seed.Limit
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Priced != rows[j].Priced {
			return rows[i].Priced
		}
		return rows[i].ProfitPerLimit > rows[j].ProfitPerLimit
	})
	return rows, nil
}
