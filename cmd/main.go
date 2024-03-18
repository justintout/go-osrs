package main

import (
	"fmt"

	"github.com/justintout/osrs/items"
)

func main() {
	c := items.NewClient()
	prices, err := c.Latest()
	if err != nil {
		panic(err)
	}
	for id, spread := range prices {
		fmt.Printf("%d:\n  Buy: %d @ %d\n  Sell: %d @ %d\n", id, spread.High, spread.HighTime, spread.Low, spread.LowTime)
	}
}
