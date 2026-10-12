package wiki

import (
	"context"
	"strconv"
	"strings"
)

// StoreItem is an item an NPC shop sells.
type StoreItem struct {
	Store string `json:"store"`
	Item  string `json:"item"`
	// Currency is usually "Coins"; other shops take tokens or points.
	Currency string `json:"currency"`
	// Price is what the shop charges for the first unit at full stock.
	Price int `json:"price"`
	// Stock is the shop's full stock, or 0 for unlimited.
	Stock int `json:"stock"`
	// SellMultiplier and Delta are in tenths of a percent of the item's
	// value: each unit bought raises the price by Delta.
	SellMultiplier int    `json:"sell_multiplier"`
	Delta          int    `json:"delta"`
	RestockTime    string `json:"restock_time,omitempty"`
	Members        bool   `json:"members"`
	Location       string `json:"location,omitempty"`
}

// Stores returns every item sold in an NPC shop.
func (c *Client) Stores(ctx context.Context) ([]StoreItem, error) {
	rows, err := runPaged[struct {
		Store    string `json:"page_name"`
		Item     string `json:"sold_item"`
		Currency string `json:"store_currency"`
		Price    string `json:"store_sell_price"`
		Stock    string `json:"store_stock"`
		Mult     string `json:"store_sell_multiplier"`
		Delta    string `json:"store_delta"`
		Restock  string `json:"restock_time"`
		JSON     string `json:"sold_item_json"`
	}](ctx, c, Bucket("storeline").Select("page_name", "sold_item", "store_currency", "store_sell_price",
		"store_stock", "store_sell_multiplier", "store_delta", "restock_time", "sold_item_json"))
	if err != nil {
		return nil, err
	}
	items := make([]StoreItem, 0, len(rows))
	for _, r := range rows {
		price, err := strconv.Atoi(strings.ReplaceAll(r.Price, ",", ""))
		if err != nil {
			// "N/A" and similar mean the shop does not sell it for a price.
			continue
		}
		stock, _ := strconv.Atoi(r.Stock) // "∞" leaves 0, meaning unlimited
		mult, _ := strconv.Atoi(r.Mult)
		delta, _ := strconv.Atoi(r.Delta)
		items = append(items, StoreItem{
			Store: r.Store, Item: r.Item, Currency: r.Currency, Price: price, Stock: stock,
			SellMultiplier: mult, Delta: delta, RestockTime: r.Restock,
			Members:  strings.Contains(r.JSON, `"Members":"Yes"`),
			Location: storeLocation(r.JSON),
		})
	}
	return items, nil
}

func storeLocation(raw string) string {
	_, after, ok := strings.Cut(raw, `"Location":"`)
	if !ok {
		return ""
	}
	if m := pageLinkRE.FindStringSubmatch(after); m != nil {
		return m[1]
	}
	return ""
}
