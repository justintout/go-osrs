// Command osrs queries Old School RuneScape hiscores, prices, the OSRS
// Wiki, and Wise Old Man, and serves the same tools over MCP.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/justintout/go-osrs/archive"
	"github.com/justintout/go-osrs/cache"
	"github.com/justintout/go-osrs/ge"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
	"github.com/justintout/go-osrs/wom"
)

// deps holds the API clients shared by every command.
type deps struct {
	hiscores *hiscores.Client
	prices   *prices.Client
	ge       *ge.Client
	wiki     *wiki.Client
	wom      *wom.Client
	// cache is nil when OSRS_NO_CACHE is set.
	cache *cache.Cache

	archiveMu sync.Mutex
	archive   *archive.Archive
}

// priceArchive opens the SQLite price archive in the cache directory on
// first use.
func (d *deps) priceArchive() (*archive.Archive, error) {
	d.archiveMu.Lock()
	defer d.archiveMu.Unlock()
	if d.archive != nil {
		return d.archive, nil
	}
	if d.cache == nil {
		return nil, errors.New("market scans keep price history in the cache directory; unset OSRS_NO_CACHE")
	}
	a, err := archive.Open(filepath.Join(d.cache.Dir(), "prices.db"), d.prices)
	if err != nil {
		return nil, err
	}
	d.archive = a
	return a, nil
}

func newDeps() (*deps, error) {
	ua := os.Getenv("OSRS_USER_AGENT")
	if ua == "" {
		ua = fmt.Sprintf("go-osrs/%s (https://github.com/justintout/go-osrs)", version())
	}
	var (
		c         *cache.Cache
		priceOpts []prices.Option
		wikiOpts  []wiki.Option
		womOpts   []wom.Option
	)
	if os.Getenv("OSRS_NO_CACHE") == "" {
		var err error
		if c, err = cache.Default(); err != nil {
			return nil, err
		}
		priceOpts, wikiOpts, womOpts = append(priceOpts, prices.WithCache(c)), append(wikiOpts, wiki.WithCache(c)), append(womOpts, wom.WithCache(c))
	}
	if key := os.Getenv("WOM_API_KEY"); key != "" {
		womOpts = append(womOpts, wom.WithAPIKey(key))
	}
	return &deps{
		hiscores: hiscores.NewClient(ua),
		prices:   prices.NewClient(ua, priceOpts...),
		ge:       ge.NewClient(ua),
		wiki:     wiki.NewClient(ua, wikiOpts...),
		wom:      wom.NewClient(ua, womOpts...),
		cache:    c,
	}, nil
}

// itemCatalog indexes the item mapping. The disk cache serves the mapping
// for an hour, as the prices API's headers allow.
func (d *deps) itemCatalog(ctx context.Context) (*prices.Catalog, error) {
	items, err := d.prices.Mapping(ctx)
	if err != nil {
		return nil, err
	}
	return prices.NewCatalog(items), nil
}

// marketData fetches the item catalog, latest prices, and 24h volumes.
func marketData(ctx context.Context, d *deps) (*prices.Catalog, prices.Prices, map[int]int, error) {
	cat, err := d.itemCatalog(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	latest, err := d.prices.Latest(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	vols, _, err := d.prices.Volumes(ctx)
	return cat, latest, vols, err
}

// findItem resolves an item name, suggesting close names when there is
// no exact match.
func (d *deps) findItem(ctx context.Context, name string) (prices.Item, error) {
	cat, err := d.itemCatalog(ctx)
	if err != nil {
		return prices.Item{}, err
	}
	if item, ok := cat.ByName(name); ok {
		return item, nil
	}
	var names []string
	for _, i := range cat.Search(name) {
		names = append(names, i.Name)
	}
	sort.Strings(names)
	if len(names) > 10 {
		names = names[:10]
	}
	if len(names) == 0 {
		return prices.Item{}, fmt.Errorf("no tradeable item named %q", name)
	}
	return prices.Item{}, fmt.Errorf("no tradeable item named %q; did you mean: %s", name, strings.Join(names, ", "))
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "osrs:", strings.TrimPrefix(err.Error(), "osrs: "))
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, w io.Writer) error {
	cmds := commands()
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage(w, cmds)
		return nil
	}
	d, err := newDeps()
	if err != nil {
		return err
	}
	switch args[0] {
	case "mcp":
		return serveMCP(ctx, d, cmds)
	case "cache":
		return cacheCmd(ctx, d, args[1:], w)
	case "portfolio":
		return portfolioCmd(ctx, d, args[1:], w)
	case "version":
		_, err := fmt.Fprintln(w, version())
		return err
	}
	for _, c := range cmds {
		if c.name == args[0] {
			return c.run(ctx, d, args[1:], w)
		}
	}
	usage(w, cmds)
	return fmt.Errorf("unknown command %q", args[0])
}

func serveMCP(ctx context.Context, d *deps, cmds []command) error {
	return newMCPServer(d, cmds).Run(ctx, &mcp.StdioTransport{})
}

func newMCPServer(d *deps, cmds []command) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "osrs", Version: version()}, nil)
	for _, c := range cmds {
		c.register(s, d)
	}
	return s
}

func usage(w io.Writer, cmds []command) {
	fmt.Fprintln(w, "osrs queries Old School RuneScape data.\n\nusage: osrs <command> [args] [--json]\n\ncommands:")
	for _, c := range cmds {
		fmt.Fprintf(w, "  %-16s %s\n", c.name, c.summary)
	}
	fmt.Fprintf(w, "  %-16s %s\n  %-16s %s\n  %-16s %s\n", "mcp", "serve every command as an MCP tool over stdio",
		"cache", "status, clear, or warm the on-disk response cache", "version", "print the version")
	fmt.Fprintf(w, "  %-16s %s\n", "portfolio", "track investments: add, fill, close, remove, sync (from RuneLite), list, check, watch, schedule")
	fmt.Fprintln(w, "\nRun \"osrs <command> -h\" for a command's flags.\nSet OSRS_USER_AGENT to identify yourself to the APIs, WOM_API_KEY for a higher Wise Old Man rate limit,\nand OSRS_NO_CACHE=1 to skip the on-disk cache.")
}
