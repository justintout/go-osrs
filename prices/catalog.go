package prices

import "strings"

// Catalog indexes item metadata from Mapping by ID and name.
type Catalog struct {
	items  []Item
	byID   map[int]Item
	byName map[string]Item
}

// NewCatalog indexes items, usually the result of Mapping.
func NewCatalog(items []Item) *Catalog {
	c := &Catalog{items: items, byID: make(map[int]Item, len(items)), byName: make(map[string]Item, len(items))}
	for _, i := range items {
		c.byID[i.ID] = i
		c.byName[strings.ToLower(i.Name)] = i
	}
	return c
}

// Items returns every item in the catalog.
func (c *Catalog) Items() []Item {
	return c.items
}

// ByID returns the item with the given ID.
func (c *Catalog) ByID(id int) (Item, bool) {
	i, ok := c.byID[id]
	return i, ok
}

// ByName returns the item with the given name, ignoring case.
func (c *Catalog) ByName(name string) (Item, bool) {
	i, ok := c.byName[strings.ToLower(name)]
	return i, ok
}

// Search returns items whose names contain query, ignoring case.
func (c *Catalog) Search(query string) []Item {
	q := strings.ToLower(query)
	var found []Item
	for _, i := range c.items {
		if strings.Contains(strings.ToLower(i.Name), q) {
			found = append(found, i)
		}
	}
	return found
}
