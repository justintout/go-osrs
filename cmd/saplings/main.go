package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.comcom/justintout/osrs/items"
)

const indexPage = `
<!DOCTYPE html>
<html>
<head>
    <title>OSRS Sapling Profit Calculator</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
    <h1>OSRS Sapling Profit Calculator</h1>
    <button hx-get="/profits" hx-target="#profits-result">
        Calculate Profits
    </button>
    <div id="profits-result"></div>
</body>
</html>
`

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
		fmt.Fprint(w, indexPage)
	})
	http.HandleFunc("/profits", func(w http.ResponseWriter, r *http.Request) {
		var results strings.Builder
		results.WriteString("<table><tr><th>Tree</th><th>Profit</th></tr>")

		for seedName, saplingName := range treePairs {
			seedID, ok := itemMap[seedName]
			if !ok {
				results.WriteString(fmt.Sprintf("<tr><td>%s</td><td>Seed not found</td></tr>", strings.TrimSuffix(seedName, " seed")))
				continue
			}
			saplingID, ok := itemMap[saplingName]
			if !ok {
				results.WriteString(fmt.Sprintf("<tr><td>%s</td><td>Sapling not found</td></tr>", strings.TrimSuffix(seedName, " seed")))
				continue
			}

			seedPrice, err := c.LatestFor(seedID)
			if err != nil {
				results.WriteString(fmt.Sprintf("<tr><td>%s</td><td>Error fetching seed price</td></tr>", strings.TrimSuffix(seedName, " seed")))
				continue
			}

			saplingPrice, err := c.LatestFor(saplingID)
			if err != nil {
				results.WriteString(fmt.Sprintf("<tr><td>%s</td><td>Error fetching sapling price</td></tr>", strings.TrimSuffix(seedName, " seed")))
				continue
			}

			profit := saplingPrice.Low - seedPrice.High
			results.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d gp</td></tr>", strings.TrimSuffix(seedName, " seed"), profit))
		}
		results.WriteString("</table>")
		fmt.Fprint(w, results.String())
	})
	http.ListenAndServe(":8080", nil)
}