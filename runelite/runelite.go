// Package runelite reads Grand Exchange offers that the RuneLite client's
// built-in Grand Exchange plugin saves in its local profile files. It only
// reads files on disk and never talks to the game.
package runelite

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Offer is one Grand Exchange slot as RuneLite last saved it.
type Offer struct {
	// Profile is RuneLite's key for the account, and Account its display
	// name, when RuneLite has recorded one.
	Profile string `json:"profile"`
	Account string `json:"account,omitempty"`
	Slot    int    `json:"slot"`
	ItemID  int    `json:"itemId"`
	// QuantitySold is how many have traded so far, for buys and sells.
	QuantitySold  int `json:"quantitySold"`
	TotalQuantity int `json:"totalQuantity"`
	// Price is the offer's limit price. Spent is the coins paid so far
	// for a buy, or received for a sell.
	Price int `json:"price"`
	Spent int `json:"spent"`
	// State is BUYING, BOUGHT, SELLING, SOLD, CANCELLED_BUY,
	// CANCELLED_SELL, or EMPTY.
	State string `json:"state"`
}

// Buy reports whether the offer is on the buy side.
func (o Offer) Buy() bool { return strings.Contains(o.State, "BUY") || o.State == "BOUGHT" }

// AveragePrice is what each traded unit actually cost or brought, which
// can beat the limit price.
func (o Offer) AveragePrice() int {
	if o.QuantitySold == 0 {
		return o.Price
	}
	return o.Spent / o.QuantitySold
}

// DefaultDir is RuneLite's profile directory, ~/.runelite/profiles2.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("runelite: %w", err)
	}
	return filepath.Join(home, ".runelite", "profiles2"), nil
}

var (
	offerKey = regexp.MustCompile(`^geoffer\.rsprofile\.([^.]+)\.(\d+)$`)
	nameKey  = regexp.MustCompile(`^rsprofile\.rsprofile\.([^.]+)\.displayName$`)
)

// Offers returns every saved Grand Exchange slot in dir, by profile and
// slot. RuneLite writes these files a short while after offers change.
func Offers(dir string) ([]Offer, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.properties"))
	if err != nil {
		return nil, fmt.Errorf("runelite: %w", err)
	}
	names := map[string]string{}
	var offers []Offer
	for _, f := range files {
		props, err := readProperties(f)
		if err != nil {
			return nil, err
		}
		for k, v := range props {
			if m := nameKey.FindStringSubmatch(k); m != nil {
				names[m[1]] = v
			}
			m := offerKey.FindStringSubmatch(k)
			if m == nil {
				continue
			}
			var o Offer
			if err := json.Unmarshal([]byte(v), &o); err != nil {
				return nil, fmt.Errorf("runelite: %s %s: %w", filepath.Base(f), k, err)
			}
			o.Profile = m[1]
			o.Slot, _ = strconv.Atoi(m[2])
			offers = append(offers, o)
		}
	}
	for i := range offers {
		offers[i].Account = names[offers[i].Profile]
	}
	sort.Slice(offers, func(i, j int) bool {
		if offers[i].Profile != offers[j].Profile {
			return offers[i].Profile < offers[j].Profile
		}
		return offers[i].Slot < offers[j].Slot
	})
	return offers, nil
}

// readProperties parses a Java .properties file's key=value lines,
// undoing backslash escapes such as "\:".
func readProperties(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("runelite: %w", err)
	}
	defer f.Close()
	props := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		props[unescape(k)] = unescape(v)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("runelite: %s: %w", path, err)
	}
	return props, nil
}

func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
