// Package craft expands an item into the full tree of recipes and raw
// materials needed to make it, as an ironman must.
package craft

import (
	"fmt"
	"sort"
	"strings"

	"github.com/justintout/go-osrs/wiki"
)

// maxDepth stops runaway expansion on unusual recipe chains.
const maxDepth = 15

// Node is an item in a crafting tree.
type Node struct {
	Item string `json:"item"`
	// Quantity is how many the parent needs, or the requested amount at
	// the root.
	Quantity float64 `json:"quantity"`
	// Recipe makes this item. It is nil for raw materials.
	Recipe *wiki.Recipe `json:"recipe,omitempty"`
	// Crafts is how many times Recipe runs to make Quantity.
	Crafts float64 `json:"crafts,omitempty"`
	// Alternatives counts other recipes that also make this item.
	Alternatives int     `json:"alternatives,omitempty"`
	Inputs       []*Node `json:"inputs,omitempty"`
}

// Graph indexes recipes by the item they make.
type Graph struct {
	byOutput map[string][]wiki.Recipe
}

// New builds a Graph from wiki.Recipes.
func New(recipes []wiki.Recipe) *Graph {
	g := &Graph{byOutput: map[string][]wiki.Recipe{}}
recipe:
	for _, r := range recipes {
		if r.Output.Quantity <= 0 {
			continue
		}
		// A recipe that consumes its own output, such as recharging,
		// cannot produce the item from scratch.
		for _, m := range r.Materials {
			if strings.EqualFold(m.Name, r.Output.Name) {
				continue recipe
			}
		}
		key := strings.ToLower(r.Output.Name)
		g.byOutput[key] = append(g.byOutput[key], r)
	}
	return g
}

// Tree expands item into everything needed to make quantity of it. When
// several recipes make an item, Tree picks the one on the item's own wiki
// page, then the one with the lowest skill requirement. An item that
// would need itself further down the tree is treated as a raw material.
func (g *Graph) Tree(item string, quantity float64) (*Node, error) {
	if _, ok := g.byOutput[strings.ToLower(item)]; !ok {
		if similar := g.similar(item); len(similar) > 0 {
			return nil, fmt.Errorf("craft: no recipe makes %q; recipes make: %s", item, strings.Join(similar, ", "))
		}
		return nil, fmt.Errorf("craft: no recipe makes %q", item)
	}
	return g.expand(item, quantity, map[string]bool{}, 0), nil
}

func (g *Graph) expand(item string, quantity float64, path map[string]bool, depth int) *Node {
	n := &Node{Item: item, Quantity: quantity}
	key := strings.ToLower(item)
	recipes := g.byOutput[key]
	if len(recipes) == 0 || path[key] || depth >= maxDepth {
		return n
	}
	r := pick(recipes, item)
	n.Recipe = &r
	n.Item = r.Output.Name
	n.Crafts = quantity / r.Output.Quantity
	n.Alternatives = len(recipes) - 1
	path[key] = true
	for _, m := range r.Materials {
		n.Inputs = append(n.Inputs, g.expand(m.Name, m.Quantity*n.Crafts, path, depth+1))
	}
	delete(path, key)
	return n
}

// similar returns up to 8 recipe outputs whose names share the item's
// first word, such as "Saradomin brew(3)" for "Saradomin brew(4)".
func (g *Graph) similar(item string) []string {
	word, _, _ := strings.Cut(strings.ToLower(item), " ")
	word = strings.TrimRight(word, "()0123456789")
	var out []string
	for _, rs := range g.byOutput {
		name := rs[0].Output.Name
		if strings.HasPrefix(strings.ToLower(name), word) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out[:min(len(out), 8)]
}

func pick(recipes []wiki.Recipe, item string) wiki.Recipe {
	best := recipes[0]
	for _, r := range recipes[1:] {
		rOwn, bestOwn := strings.EqualFold(r.Page, item), strings.EqualFold(best.Page, item)
		if (rOwn && !bestOwn) || (rOwn == bestOwn && maxLevel(r) < maxLevel(best)) {
			best = r
		}
	}
	return best
}

func maxLevel(r wiki.Recipe) int {
	m := 0
	for _, s := range r.Skills {
		m = max(m, s.Level)
	}
	return m
}

// Material is a raw material and the total amount the tree needs.
type Material struct {
	Item     string  `json:"item"`
	Quantity float64 `json:"quantity"`
}

// Raw sums the raw materials at the leaves of the tree, largest first.
func (n *Node) Raw() []Material {
	totals := map[string]float64{}
	var walk func(*Node)
	walk = func(n *Node) {
		if n.Recipe == nil {
			totals[n.Item] += n.Quantity
		}
		for _, in := range n.Inputs {
			walk(in)
		}
	}
	walk(n)
	out := make([]Material, 0, len(totals))
	for item, q := range totals {
		out = append(out, Material{Item: item, Quantity: q})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Quantity != out[j].Quantity {
			return out[i].Quantity > out[j].Quantity
		}
		return out[i].Item < out[j].Item
	})
	return out
}

// SkillTotal is a skill's highest level requirement in the tree and the
// XP gained making everything.
type SkillTotal struct {
	Skill string  `json:"skill"`
	Level int     `json:"level"`
	XP    float64 `json:"xp"`
}

// Skills returns each skill the tree uses, by name.
func (n *Node) Skills() []SkillTotal {
	totals := map[string]*SkillTotal{}
	var walk func(*Node)
	walk = func(n *Node) {
		if n.Recipe != nil {
			for _, s := range n.Recipe.Skills {
				t, ok := totals[s.Name]
				if !ok {
					t = &SkillTotal{Skill: s.Name}
					totals[s.Name] = t
				}
				t.Level = max(t.Level, s.Level)
				t.XP += s.XP * n.Crafts
			}
		}
		for _, in := range n.Inputs {
			walk(in)
		}
	}
	walk(n)
	out := make([]SkillTotal, 0, len(totals))
	for _, t := range totals {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Skill < out[j].Skill })
	return out
}

// Text draws the tree with box-drawing branches.
func (n *Node) Text() string {
	var b strings.Builder
	var walk func(n *Node, prefix string, last, root bool)
	walk = func(n *Node, prefix string, last, root bool) {
		line, child := "", prefix
		if !root {
			branch := "├── "
			child += "│   "
			if last {
				branch, child = "└── ", prefix+"    "
			}
			line = prefix + branch
		}
		fmt.Fprintf(&b, "%s%s x%s%s\n", line, n.Item, qty(n.Quantity), n.label())
		for i, in := range n.Inputs {
			walk(in, child, i == len(n.Inputs)-1, false)
		}
	}
	walk(n, "", true, true)
	return b.String()
}

// Mermaid draws the tree as a Mermaid flowchart, with arrows from each
// input to what it makes.
func (n *Node) Mermaid() string {
	var b strings.Builder
	b.WriteString("flowchart BT\n")
	id := 0
	var walk func(n *Node) string
	walk = func(n *Node) string {
		me := fmt.Sprintf("n%d", id)
		id++
		shape := `%s["%s x%s%s"]`
		if n.Recipe == nil {
			shape = `%s(["%s x%s%s"])` // raw materials are rounded
		}
		fmt.Fprintf(&b, "  "+shape+"\n", me, mermaidEscape(n.Item), qty(n.Quantity), mermaidEscape(n.label()))
		for _, in := range n.Inputs {
			fmt.Fprintf(&b, "  %s --> %s\n", walk(in), me)
		}
		return me
	}
	walk(n)
	return b.String()
}

func (n *Node) label() string {
	if n.Recipe == nil {
		return ""
	}
	var parts []string
	for _, s := range n.Recipe.Skills {
		parts = append(parts, fmt.Sprintf("%d %s", s.Level, s.Name))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func mermaidEscape(s string) string {
	return strings.ReplaceAll(s, `"`, "#quot;")
}

// qty prints whole numbers plainly and fractions to two places.
func qty(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprint(int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}
