package prices

// TaxCap is the most tax the Grand Exchange takes from one sale.
const TaxCap = 5_000_000

// Tax returns the Grand Exchange tax on selling one of an item at price:
// 2%, rounded down, capped at TaxCap. Rounding down exempts prices under
// 50. Some items are exempt at any price.
// https://oldschool.runescape.wiki/w/Grand_Exchange#Convenience_fee_and_item_sink
func Tax(itemID, price int) int {
	if taxExempt[itemID] {
		return 0
	}
	return min(price/50, TaxCap)
}

// AfterTax returns what the seller receives for one of an item sold at
// price.
func AfterTax(itemID, price int) int {
	return price - Tax(itemID, price)
}
