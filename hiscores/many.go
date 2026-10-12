package hiscores

import (
	"context"
	"sync"
)

// Result is one player's outcome from LookupMany.
type Result struct {
	Name   string  `json:"name"`
	Player *Player `json:"player,omitempty"`
	Err    error   `json:"-"`
}

// LookupMany looks up players on one mode's hiscores with at most
// concurrency requests in flight, and returns results in input order.
// A failed lookup sets that result's Err and does not stop the others.
func (c *Client) LookupMany(ctx context.Context, players []string, mode Mode, concurrency int) []Result {
	results := make([]Result, len(players))
	sem := make(chan struct{}, max(concurrency, 1))
	var wg sync.WaitGroup
	for i, name := range players {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			p, err := c.Lookup(ctx, name, mode)
			results[i] = Result{Name: name, Player: p, Err: err}
		}()
	}
	wg.Wait()
	return results
}
