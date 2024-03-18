package main

import (
	"fmt"
	"time"

	"github.com/justintout/osrs/items"
)

func main() {
	c := items.NewClient("go-osrs Test Command (https://github.com/justintout/go-osrs/cmd)")
	prices, err := c.Latest()
	if err != nil {
		panic(err)
	}
	for id, spread := range prices {
		fmt.Printf("%d:\n  Buy: %d @ %s\n  Sell: %d @ %s\n", id, spread.High, spread.HighTime.Format(time.RFC3339), spread.Low, spread.LowTime.Format(time.RFC3339))
	}
}
