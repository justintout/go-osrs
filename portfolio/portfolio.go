// Package portfolio tracks Grand Exchange investments and checks each one
// against its plan: target price, exit window, drawdown, unfilled offers,
// and game updates that could change the thesis.
package portfolio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Position is one investment: a buy offer, a filled holding, or a closed
// trade.
type Position struct {
	// ID is a short name such as "zenyte-1".
	ID     string `json:"id"`
	Item   string `json:"item"`
	ItemID int    `json:"item_id"`
	// Quantity is how many were offered for; Filled is how many bought.
	Quantity int `json:"quantity"`
	Filled   int `json:"filled"`
	// Price is the average fill price, or the offer price before fills.
	Price int `json:"price"`
	// OfferPrice is the buy offer's limit price, which identifies the
	// offer when syncing from RuneLite.
	OfferPrice int       `json:"offer_price,omitempty"`
	Opened     time.Time `json:"opened"`
	// Target is the planned sell price per unit, before tax.
	Target int `json:"target,omitempty"`
	// ExitFrom and ExitBy bound when the plan says to sell.
	ExitFrom time.Time `json:"exit_from,omitempty"`
	ExitBy   time.Time `json:"exit_by,omitempty"`
	// MaxDrawdown is the after-tax loss, as a fraction of cost, that
	// calls for a review. 0 means 0.15.
	MaxDrawdown float64   `json:"max_drawdown,omitempty"`
	Note        string    `json:"note,omitempty"`
	ClosedPrice int       `json:"closed_price,omitempty"`
	Closed      time.Time `json:"closed,omitempty"`
}

// Open reports whether the position is still held or offered for.
func (p Position) Open() bool { return p.Closed.IsZero() }

// Book is the saved set of positions and the alerts already sent.
type Book struct {
	Positions []Position `json:"positions"`
	// Sent maps alert keys to when they were first sent, so watchers do
	// not repeat themselves.
	Sent map[string]time.Time `json:"sent,omitempty"`
}

// DefaultPath is portfolio.json in the user's config directory, such as
// ~/Library/Application Support/go-osrs on macOS.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("portfolio: %w", err)
	}
	return filepath.Join(dir, "go-osrs", "portfolio.json"), nil
}

// Load reads the book at path. A missing file is an empty book.
func Load(path string) (*Book, error) {
	b := &Book{Sent: map[string]time.Time{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return b, nil
	}
	if err != nil {
		return nil, fmt.Errorf("portfolio: %w", err)
	}
	if err := json.Unmarshal(data, b); err != nil {
		return nil, fmt.Errorf("portfolio: %s: %w", path, err)
	}
	if b.Sent == nil {
		b.Sent = map[string]time.Time{}
	}
	return b, nil
}

// Save writes the book to path atomically.
func (b *Book) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("portfolio: %w", err)
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("portfolio: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("portfolio: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("portfolio: %w", err)
	}
	return nil
}

// Find returns the position with the given ID.
func (b *Book) Find(id string) (*Position, error) {
	for i := range b.Positions {
		if strings.EqualFold(b.Positions[i].ID, id) {
			return &b.Positions[i], nil
		}
	}
	return nil, fmt.Errorf("portfolio: no position %q", id)
}

// Add appends p, giving it an ID from the item name if it has none.
func (b *Book) Add(p Position) (*Position, error) {
	if p.OfferPrice == 0 {
		p.OfferPrice = p.Price
	}
	if p.ID == "" {
		base := strings.ToLower(strings.Fields(p.Item)[0])
		p.ID = base
		for n := 2; ; n++ {
			if _, err := b.Find(p.ID); err != nil {
				break
			}
			p.ID = fmt.Sprintf("%s-%d", base, n)
		}
	}
	if _, err := b.Find(p.ID); err == nil {
		return nil, fmt.Errorf("portfolio: position %q already exists", p.ID)
	}
	b.Positions = append(b.Positions, p)
	sort.SliceStable(b.Positions, func(i, j int) bool { return b.Positions[i].Opened.Before(b.Positions[j].Opened) })
	return b.Find(p.ID)
}

// Remove deletes the position with the given ID.
func (b *Book) Remove(id string) error {
	for i, p := range b.Positions {
		if strings.EqualFold(p.ID, id) {
			b.Positions = append(b.Positions[:i], b.Positions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("portfolio: no position %q", id)
}
