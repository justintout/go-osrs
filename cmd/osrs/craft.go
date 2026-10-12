package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/justintout/go-osrs/craft"
	"github.com/justintout/go-osrs/wiki"
)

type craftIn struct {
	Item     string `json:"item" arg:"" jsonschema:"item to make, such as Saradomin brew(4)"`
	Quantity int    `json:"quantity,omitempty" jsonschema:"how many to make, 1 by default"`
	Format   string `json:"format,omitempty" jsonschema:"text (default) or mermaid, a flowchart diagram"`
}

type rawLine struct {
	Item     string  `json:"item"`
	Quantity float64 `json:"quantity"`
	// Sources are ways an ironman can get it: gathering skills, the
	// cheapest coin shop, or the best drop when neither exists.
	Sources []string `json:"sources"`
}

type craftOut struct {
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
	// Tree holds a *craft.Node. It is typed any because the MCP schema
	// generator cannot describe a type that contains itself.
	Tree    any                `json:"tree"`
	Raw     []rawLine          `json:"raw_materials"`
	Skills  []craft.SkillTotal `json:"skills"`
	Mermaid string             `json:"mermaid"`
	format  string
}

func (o craftOut) text() string {
	if o.format == "mermaid" {
		return o.Mermaid
	}
	var b strings.Builder
	b.WriteString(o.Tree.(*craft.Node).Text())
	b.WriteString("\nraw materials:\n")
	t := newTable(&b, "ITEM", "QUANTITY", "HOW TO GET")
	for _, r := range o.Raw {
		t.row(r.Item, fmtQty(r.Quantity), strings.Join(r.Sources, "; "))
	}
	t.flush()
	if len(o.Skills) > 0 {
		b.WriteString("\nskills:\n")
		t := newTable(&b, "SKILL", "HIGHEST LEVEL", "XP GAINED")
		for _, s := range o.Skills {
			t.row(s.Skill, fmt.Sprint(s.Level), num(int64(s.XP)))
		}
		t.flush()
	}
	return b.String()
}

var craftCmd = newCommand("craft", "expand an item into its full crafting tree down to raw materials, with how an ironman gets each one, the skills needed, and a Mermaid diagram",
	func(ctx context.Context, d *deps, in craftIn) (craftOut, error) {
		if err := need("item", in.Item); err != nil {
			return craftOut{}, err
		}
		if in.Format != "" && in.Format != "text" && in.Format != "mermaid" {
			return craftOut{}, fmt.Errorf("format must be text or mermaid")
		}
		recipes, err := d.wiki.Recipes(ctx)
		if err != nil {
			return craftOut{}, err
		}
		qty := orDefault(in.Quantity, 1)
		tree, err := craft.New(recipes).Tree(in.Item, float64(qty))
		if err != nil {
			return craftOut{}, err
		}
		raw, err := rawSources(ctx, d, tree.Raw())
		if err != nil {
			return craftOut{}, err
		}
		return craftOut{Item: tree.Item, Quantity: qty, Tree: tree, Raw: raw, Skills: tree.Skills(), Mermaid: tree.Mermaid(), format: in.Format}, nil
	})

// gatherCategories maps wiki item categories to how the item is gathered.
var gatherCategories = []struct{ category, how string }{
	{"Woodcutting", "gather with Woodcutting"},
	{"Logs", "gather with Woodcutting"},
	{"Mining", "gather with Mining"},
	{"Ores", "gather with Mining"},
	{"Fishing", "gather with Fishing"},
	{"Fish", "gather with Fishing"},
	{"Hunter", "gather with Hunter"},
	{"Farming", "grow with Farming"},
	{"Herbs", "grow with Farming"},
}

// rawSources finds how to get each raw material: gathering skills from
// the item page's categories, the cheapest coin shop, and, for items with
// neither, the most likely drop.
func rawSources(ctx context.Context, d *deps, materials []craft.Material) ([]rawLine, error) {
	names := make([]string, len(materials))
	for i, m := range materials {
		names[i] = m.Item
	}
	cats, err := d.wiki.Categories(ctx, names)
	if err != nil {
		return nil, err
	}
	stores, err := d.wiki.Stores(ctx)
	if err != nil {
		return nil, err
	}
	shops := map[string]wiki.StoreItem{}
	for _, s := range stores {
		cur, ok := shops[s.Item]
		if s.Currency == "Coins" && s.Stock != 0 && (!ok || s.Price < cur.Price) {
			shops[s.Item] = s
		}
	}

	lines := make([]rawLine, len(materials))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	errs := make([]error, len(materials))
	for i, m := range materials {
		l := rawLine{Item: m.Item, Quantity: m.Quantity}
		for _, g := range gatherCategories {
			if slices.Contains(cats[m.Item], g.category) && !slices.Contains(l.Sources, g.how) {
				l.Sources = append(l.Sources, g.how)
			}
		}
		if s, ok := shops[m.Item]; ok {
			where := s.Store
			if s.Location != "" {
				where += ", " + s.Location
			}
			l.Sources = append(l.Sources, fmt.Sprintf("buy at %s for %s gp", where, num(int64(s.Price))))
		}
		lines[i] = l
		if len(l.Sources) > 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			drops, err := d.wiki.DropSources(ctx, m.Item)
			if err != nil {
				errs[i] = err
				return
			}
			var best *wiki.Drop
			for j, dr := range drops {
				if best == nil || dr.Probability > best.Probability {
					best = &drops[j]
				}
			}
			if best != nil {
				lines[i].Sources = append(lines[i].Sources, fmt.Sprintf("drop from %s (%s)", best.Source, best.Rarity))
			} else {
				lines[i].Sources = append(lines[i].Sources, "see the wiki")
			}
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func fmtQty(q float64) string {
	if q == float64(int64(q)) {
		return num(int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}
