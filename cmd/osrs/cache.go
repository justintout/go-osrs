package main

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// cacheCmd manages the on-disk cache. It is CLI-only: clearing a shared
// cache is not something an MCP client should do.
func cacheCmd(ctx context.Context, d *deps, args []string, w io.Writer) error {
	if d.cache == nil {
		return errors.New("the cache is off because OSRS_NO_CACHE is set")
	}
	if len(args) != 1 {
		return errors.New("usage: osrs cache status|clear|warm")
	}
	switch args[0] {
	case "status":
		s, err := d.cache.Stats()
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\n%d entries, %d fresh, %.1f MB\n", d.cache.Dir(), s.Entries, s.Fresh, float64(s.Bytes)/1e6)
		return nil
	case "clear":
		if err := d.cache.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(w, "cleared", d.cache.Dir())
		return nil
	case "warm":
		return warm(ctx, d, w)
	}
	return fmt.Errorf("unknown cache command %q", args[0])
}

// warm fetches the large tables that scans use, so later commands read
// them from disk.
func warm(ctx context.Context, d *deps, w io.Writer) error {
	steps := []struct {
		name string
		run  func() error
	}{
		{"item mapping", func() error { _, err := d.prices.Mapping(ctx); return err }},
		{"recipes", func() error { _, err := d.wiki.Recipes(ctx); return err }},
		{"drop tables", func() error { _, err := d.wiki.AllDrops(ctx); return err }},
		{"shops", func() error { _, err := d.wiki.Stores(ctx); return err }},
		{"collection log", func() error { _, err := d.wiki.CollectionLog(ctx); return err }},
		{"quests", func() error { _, err := d.wiki.Quests(ctx); return err }},
		{"updates", func() error { _, err := d.wiki.Updates(ctx); return err }},
	}
	for _, s := range steps {
		fmt.Fprintf(w, "%s... ", s.name)
		if err := s.run(); err != nil {
			fmt.Fprintln(w)
			return fmt.Errorf("warm %s: %w", s.name, err)
		}
		fmt.Fprintln(w, "ok")
	}
	return nil
}
