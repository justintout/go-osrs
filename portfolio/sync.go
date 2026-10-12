package portfolio

import (
	"fmt"
	"time"

	"github.com/justintout/go-osrs/runelite"
)

// Change describes what a sync changed or found.
type Change struct {
	Position string `json:"position,omitempty"`
	Message  string `json:"message"`
	// Untracked is set for a buy offer that matches no position.
	Untracked *runelite.Offer `json:"untracked,omitempty"`
}

// Sync updates positions from RuneLite's saved Grand Exchange offers.
//
// A buy offer matches an open position with the same item, offer price,
// and quantity; its fills and average price are copied over. Fills only
// grow, since RuneLite saves its files a while after offers change and a
// count entered by hand may be newer. A completed sell offer for a held
// item closes the position at the sale's average price. Buy offers that
// match nothing are reported as untracked.
func Sync(b *Book, offers []runelite.Offer, now time.Time) []Change {
	var changes []Change
	for _, o := range offers {
		if o.State == "EMPTY" {
			continue
		}
		if o.Buy() {
			p := matchBuy(b, o)
			if p == nil {
				changes = append(changes, Change{Message: fmt.Sprintf("untracked buy offer: item %d, %d at %s (slot %d, %s)", o.ItemID, o.TotalQuantity, gp(o.Price), o.Slot, o.State), Untracked: &o})
				continue
			}
			if o.QuantitySold > p.Filled {
				p.Filled, p.Price = o.QuantitySold, o.AveragePrice()
				changes = append(changes, Change{Position: p.ID, Message: fmt.Sprintf("%s: %d/%d filled at an average %s", p.ID, p.Filled, p.Quantity, gp(p.Price))})
			}
			continue
		}
		if o.State != "SOLD" {
			continue
		}
		for i := range b.Positions {
			p := &b.Positions[i]
			if p.Open() && p.ItemID == o.ItemID && p.Filled > 0 && o.QuantitySold >= p.Filled {
				p.ClosedPrice, p.Closed = o.AveragePrice(), now
				changes = append(changes, Change{Position: p.ID, Message: fmt.Sprintf("%s: sold %d at an average %s; closed", p.ID, o.QuantitySold, gp(p.ClosedPrice))})
				break
			}
		}
	}
	return changes
}

func matchBuy(b *Book, o runelite.Offer) *Position {
	for i := range b.Positions {
		p := &b.Positions[i]
		if p.Open() && p.ItemID == o.ItemID && p.OfferPrice == o.Price && p.Quantity == o.TotalQuantity {
			return p
		}
	}
	return nil
}
