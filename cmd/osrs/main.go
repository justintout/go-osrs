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
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

	catalogMu sync.Mutex
	catalog   *prices.Catalog
}

func newDeps() *deps {
	ua := os.Getenv("OSRS_USER_AGENT")
	if ua == "" {
		ua = fmt.Sprintf("go-osrs/%s (https://github.com/justintout/go-osrs)", version())
	}
	var womOpts []wom.Option
	if key := os.Getenv("WOM_API_KEY"); key != "" {
		womOpts = append(womOpts, wom.WithAPIKey(key))
	}
	return &deps{
		hiscores: hiscores.NewClient(ua),
		prices:   prices.NewClient(ua),
		ge:       ge.NewClient(ua),
		wiki:     wiki.NewClient(ua),
		wom:      wom.NewClient(ua, womOpts...),
	}
}

// itemCatalog loads the wiki item mapping once per process, so the MCP
// server reuses it across tool calls. A failed load is retried next call.
func (d *deps) itemCatalog(ctx context.Context) (*prices.Catalog, error) {
	d.catalogMu.Lock()
	defer d.catalogMu.Unlock()
	if d.catalog == nil {
		items, err := d.prices.Mapping(ctx)
		if err != nil {
			return nil, err
		}
		d.catalog = prices.NewCatalog(items)
	}
	return d.catalog, nil
}

// market fetches the item catalog, latest prices, and 24h volumes.
func market(ctx context.Context, d *deps) (*prices.Catalog, prices.Prices, map[int]int, error) {
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
	d := newDeps()
	if args[0] == "mcp" {
		return serveMCP(ctx, d, cmds)
	}
	if args[0] == "version" {
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
	fmt.Fprintf(w, "  %-16s %s\n  %-16s %s\n", "mcp", "serve every command as an MCP tool over stdio", "version", "print the version")
	fmt.Fprintln(w, "\nRun \"osrs <command> -h\" for a command's flags.\nSet OSRS_USER_AGENT to identify yourself to the APIs and WOM_API_KEY for a higher Wise Old Man rate limit.")
}
