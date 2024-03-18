package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/justintout/osrs/items"
)

func main() {
	c := items.NewClient("go-osrs/0.0.1 (https://github.com/justintout/go-osrs/cmd)")
	// prices, err := c.Latest()
	// if err != nil {
	// 	panic(err)
	// }
	// for id, spread := range prices {
	// 	fmt.Printf("%d:\n  Buy: %d @ %s\n  Sell: %d @ %s\n", id, spread.High, spread.HighTime.Format(time.RFC3339), spread.Low, spread.LowTime.Format(time.RFC3339))
	// }

	items, err := c.Mapping()
	if err != nil {
		panic(err)
	}
	var id int
	for _, i := range items {
		fmt.Println(i.Name)
		if strings.EqualFold(i.Name, "Ale of the gods") {
			id = i.ID
		}
	}
	if id == 0 {
		panic("no item found")
	}
	spread, err := c.LatestFor(id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Ale of the gods: %d @ %s\n", spread.Low, spread.LowTime.Format(time.RFC3339))
}
