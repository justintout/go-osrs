package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/justintout/go-osrs/portfolio"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/runelite"
	"github.com/justintout/go-osrs/wiki"
)

func portfolioPath() (string, error) {
	if p := os.Getenv("OSRS_PORTFOLIO"); p != "" {
		return p, nil
	}
	return portfolio.DefaultPath()
}

// syncRuneLite updates b from RuneLite's saved offers when RuneLite's
// profile directory exists, and reports what changed. OSRS_ACCOUNT limits
// it to one account's display name.
func syncRuneLite(ctx context.Context, d *deps, b *portfolio.Book, w io.Writer) error {
	dir := os.Getenv("RUNELITE_PROFILES")
	if dir == "" {
		var err error
		if dir, err = runelite.DefaultDir(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	offers, err := runelite.Offers(dir)
	if err != nil {
		return err
	}
	if acct := os.Getenv("OSRS_ACCOUNT"); acct != "" {
		var kept []runelite.Offer
		for _, o := range offers {
			if strings.EqualFold(o.Account, acct) {
				kept = append(kept, o)
			}
		}
		offers = kept
	}
	cat, err := d.itemCatalog(ctx)
	if err != nil {
		return err
	}
	for _, c := range portfolio.Sync(b, offers, time.Now().UTC()) {
		msg := c.Message
		if o := c.Untracked; o != nil {
			name := fmt.Sprintf("item %d", o.ItemID)
			if it, ok := cat.ByID(o.ItemID); ok {
				name = it.Name
			}
			msg = fmt.Sprintf("untracked offer on %s: buy %d %s at %s (%s); add it with osrs portfolio add", orDefaultString(o.Account, o.Profile), o.TotalQuantity, name, num(int64(o.Price)), strings.ToLower(o.State))
		}
		fmt.Fprintln(w, "runelite:", msg)
	}
	return nil
}

// checkBook evaluates every open position against live prices and game
// updates that mention its item.
func checkBook(ctx context.Context, d *deps, b *portfolio.Book) ([]portfolio.Report, error) {
	latest, err := d.prices.Latest(ctx)
	if err != nil {
		return nil, err
	}
	updates, err := d.wiki.Updates(ctx)
	if err != nil {
		return nil, err
	}
	dates := map[string]wiki.Update{}
	for _, u := range updates {
		dates[u.Title] = u
	}
	mentions := map[string][]wiki.Update{}
	var reports []portfolio.Report
	for _, p := range b.Positions {
		if !p.Open() {
			continue
		}
		if _, ok := mentions[p.Item]; !ok {
			hits, err := d.wiki.SearchWith(ctx, wiki.SearchQuery{Query: `"` + p.Item + `"`, Namespace: wiki.NamespaceUpdate, Sort: "create_timestamp_desc", Limit: 10})
			if err != nil {
				return nil, err
			}
			mentions[p.Item] = []wiki.Update{}
			for _, h := range hits {
				if u, ok := dates[strings.TrimPrefix(h.Title, "Update:")]; ok {
					mentions[p.Item] = append(mentions[p.Item], u)
				}
			}
		}
		reports = append(reports, portfolio.Evaluate(p, latest[p.ItemID], mentions[p.Item], time.Now()))
	}
	return reports, nil
}

type portfolioOut struct {
	Reports []portfolio.Report `json:"reports"`
	Cost    int                `json:"cost"`
	Value   int                `json:"value"`
	PnL     int                `json:"pnl"`
}

func (o portfolioOut) text() string {
	var b strings.Builder
	if len(o.Reports) == 0 {
		return "no open positions; add one with: osrs portfolio add <item> --quantity N --price P\n"
	}
	t := newTable(&b, "ID", "ITEM", "FILLED", "PRICE", "SELL NOW", "TARGET", "EXIT WINDOW", "P&L AFTER TAX")
	for _, r := range o.Reports {
		p := r.Position
		window := "-"
		if !p.ExitFrom.IsZero() {
			window = p.ExitFrom.Format("Jan 02") + " - " + p.ExitBy.Format("Jan 02")
		}
		pnl := "-"
		if p.Filled > 0 {
			pnl = fmt.Sprintf("%s (%+.1f%%)", signed(r.PnL), r.PnLPct*100)
		}
		t.row(p.ID, p.Item, fmt.Sprintf("%d/%d", p.Filled, p.Quantity), num(int64(p.Price)), num(int64(r.Sell)), priceCell(p.Target), window, pnl)
	}
	t.flush()
	fmt.Fprintf(&b, "\nfilled cost %s, value after tax %s, P&L %s\n", num(int64(o.Cost)), num(int64(o.Value)), signed(o.PnL))
	var alerts []string
	for _, r := range o.Reports {
		for _, a := range r.Alerts {
			alerts = append(alerts, fmt.Sprintf("[%s] %s", a.Kind, a.Message))
		}
	}
	if len(alerts) > 0 {
		b.WriteString("\nalerts:\n  " + strings.Join(alerts, "\n  ") + "\n")
	}
	return b.String()
}

func summarize(reports []portfolio.Report) portfolioOut {
	out := portfolioOut{Reports: reports}
	for _, r := range reports {
		out.Cost += r.Cost
		out.Value += r.Value
		out.PnL += r.PnL
	}
	return out
}

// portfolioCheckCmd is the read-only check, also served over MCP.
var portfolioCheckCmd = newCommand("portfolio-check", "check saved Grand Exchange investments against their plans: P&L after tax, targets, exit windows, drawdowns, unfilled offers, and game updates naming the items",
	func(ctx context.Context, d *deps, _ struct{}) (portfolioOut, error) {
		path, err := portfolioPath()
		if err != nil {
			return portfolioOut{}, err
		}
		b, err := portfolio.Load(path)
		if err != nil {
			return portfolioOut{}, err
		}
		reports, err := checkBook(ctx, d, b)
		return summarize(reports), err
	})

type addIn struct {
	Item     string  `json:"item" arg:"" jsonschema:"item name"`
	Quantity int     `json:"quantity" jsonschema:"units offered for"`
	Filled   int     `json:"filled,omitempty" jsonschema:"units already bought"`
	Price    int     `json:"price" jsonschema:"offer or average fill price per unit"`
	Target   int     `json:"target,omitempty" jsonschema:"planned sell price per unit"`
	ExitFrom string  `json:"exit_from,omitempty" jsonschema:"start of the exit window, YYYY-MM-DD"`
	ExitBy   string  `json:"exit_by,omitempty" jsonschema:"end of the exit window, YYYY-MM-DD"`
	Drawdown float64 `json:"drawdown,omitempty" jsonschema:"after-tax loss fraction that triggers a review, 0.15 by default"`
	Opened   string  `json:"opened,omitempty" jsonschema:"date the offer was placed, YYYY-MM-DD; today by default"`
	ID       string  `json:"id,omitempty" jsonschema:"short name; derived from the item by default"`
	Note     string  `json:"note,omitempty" jsonschema:"why you made this investment"`
}

type fillIn struct {
	ID       string `json:"id" arg:"" jsonschema:"position ID"`
	Quantity int    `json:"quantity" jsonschema:"total units filled so far"`
	Price    int    `json:"price,omitempty" jsonschema:"average fill price, if different from the offer"`
}

type closeIn struct {
	ID    string `json:"id" arg:"" jsonschema:"position ID"`
	Price int    `json:"price" jsonschema:"average sale price per unit"`
}

type idOnly struct {
	ID string `json:"id" arg:"" jsonschema:"position ID"`
}

type watchIn struct {
	Every  string `json:"every,omitempty" jsonschema:"interval between checks, 6h by default and at least 15m"`
	Notify string `json:"notify,omitempty" jsonschema:"desktop for a system notification, or a webhook URL (Discord-compatible); print only by default"`
	Once   bool   `json:"once,omitempty" jsonschema:"check once, send new alerts, and exit; for cron or launchd"`
}

// portfolioCmd handles "osrs portfolio <sub>". Commands that change the
// book are CLI-only.
func portfolioCmd(ctx context.Context, d *deps, args []string, w io.Writer) error {
	const usage = "usage: osrs portfolio add|fill|close|remove|sync|list|check|watch|schedule [args]"
	if len(args) == 0 {
		return errors.New(usage)
	}
	path, err := portfolioPath()
	if err != nil {
		return err
	}
	b, err := portfolio.Load(path)
	if err != nil {
		return err
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		var in addIn
		if _, err := parseFlags("portfolio add", "add a buy offer or holding", &in, rest); err != nil {
			return err
		}
		if in.Quantity <= 0 || in.Price <= 0 {
			return errors.New("quantity and price are required")
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return err
		}
		p := portfolio.Position{ID: in.ID, Item: item.Name, ItemID: item.ID, Quantity: in.Quantity, Filled: in.Filled, Price: in.Price,
			Target: in.Target, MaxDrawdown: in.Drawdown, Note: in.Note, Opened: time.Now().UTC()}
		for _, f := range []struct {
			s   string
			dst *time.Time
		}{{in.Opened, &p.Opened}, {in.ExitFrom, &p.ExitFrom}, {in.ExitBy, &p.ExitBy}} {
			if f.s == "" {
				continue
			}
			t, err := time.Parse(time.DateOnly, f.s)
			if err != nil {
				return fmt.Errorf("dates are YYYY-MM-DD: %w", err)
			}
			*f.dst = t
		}
		added, err := b.Add(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "added %s: %d %s at %s\n", added.ID, added.Quantity, added.Item, num(int64(added.Price)))
	case "fill":
		var in fillIn
		if _, err := parseFlags("portfolio fill", "record units filled", &in, rest); err != nil {
			return err
		}
		p, err := b.Find(in.ID)
		if err != nil {
			return err
		}
		if in.Quantity < 0 || in.Quantity > p.Quantity {
			return fmt.Errorf("filled must be between 0 and %d", p.Quantity)
		}
		p.Filled = in.Quantity
		if in.Price > 0 {
			p.Price = in.Price
		}
		fmt.Fprintf(w, "%s: %d/%d filled at %s\n", p.ID, p.Filled, p.Quantity, num(int64(p.Price)))
	case "close":
		var in closeIn
		if _, err := parseFlags("portfolio close", "record a sale", &in, rest); err != nil {
			return err
		}
		p, err := b.Find(in.ID)
		if err != nil {
			return err
		}
		p.ClosedPrice, p.Closed = in.Price, time.Now().UTC()
		profit := p.Filled*prices.AfterTax(p.ItemID, in.Price) - p.Filled*p.Price
		fmt.Fprintf(w, "closed %s: %d at %s, profit after tax about %s\n", p.ID, p.Filled, num(int64(in.Price)), signed(profit))
	case "remove":
		var in idOnly
		if _, err := parseFlags("portfolio remove", "delete a position", &in, rest); err != nil {
			return err
		}
		if err := b.Remove(in.ID); err != nil {
			return err
		}
		fmt.Fprintln(w, "removed", in.ID)
	case "sync":
		if err := syncRuneLite(ctx, d, b, w); err != nil {
			return err
		}
	case "list", "check":
		if err := syncRuneLite(ctx, d, b, w); err != nil {
			return err
		}
		if err := b.Save(path); err != nil {
			return err
		}
		reports, err := checkBook(ctx, d, b)
		if err != nil {
			return err
		}
		out := summarize(reports)
		if len(rest) > 0 && rest[0] == "--json" {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		_, err = io.WriteString(w, out.text())
		return err
	case "watch":
		var in watchIn
		if _, err := parseFlags("portfolio watch", "check on an interval and send new alerts", &in, rest); err != nil {
			return err
		}
		return watch(ctx, d, path, in, w)
	case "schedule":
		var in watchIn
		if _, err := parseFlags("portfolio schedule", "print a launchd or cron entry that runs checks", &in, rest); err != nil {
			return err
		}
		return printSchedule(in, w)
	default:
		return errors.New(usage)
	}
	return b.Save(path)
}

// watch checks the book every interval and sends alerts it has not sent.
func watch(ctx context.Context, d *deps, path string, in watchIn, w io.Writer) error {
	every, err := time.ParseDuration(orDefaultString(in.Every, "6h"))
	if err != nil || every < 15*time.Minute {
		return errors.New("every must be a duration of at least 15m, such as 6h")
	}
	for {
		if err := checkAndNotify(ctx, d, path, in.Notify, w); err != nil {
			fmt.Fprintln(w, "check failed:", err)
		}
		if in.Once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(every):
		}
	}
}

func checkAndNotify(ctx context.Context, d *deps, path, notify string, w io.Writer) error {
	b, err := portfolio.Load(path)
	if err != nil {
		return err
	}
	if err := syncRuneLite(ctx, d, b, w); err != nil {
		return err
	}
	reports, err := checkBook(ctx, d, b)
	if err != nil {
		return err
	}
	var fresh []string
	for _, r := range reports {
		for _, a := range r.Alerts {
			if _, sent := b.Sent[a.Key]; !sent {
				fresh = append(fresh, a.Message)
				b.Sent[a.Key] = time.Now().UTC()
			}
		}
	}
	out := summarize(reports)
	fmt.Fprintf(w, "%s  %d open, P&L %s, %d new alerts\n", time.Now().Format(time.DateTime), len(reports), signed(out.PnL), len(fresh))
	for _, m := range fresh {
		fmt.Fprintln(w, "  "+m)
	}
	if len(fresh) > 0 && notify != "" {
		if err := send(ctx, notify, "osrs portfolio", strings.Join(fresh, "\n")); err != nil {
			return err
		}
	}
	return b.Save(path)
}

// send delivers a notification to the desktop or a webhook.
func send(ctx context.Context, target, title, body string) error {
	if target != "desktop" {
		payload, _ := json.Marshal(map[string]string{"content": "**" + title + "**\n" + body})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode > 299 {
			return fmt.Errorf("webhook returned %s", res.Status)
		}
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %q with title %q", body, title)
		return exec.CommandContext(ctx, "osascript", "-e", script).Run()
	case "linux":
		return exec.CommandContext(ctx, "notify-send", title, body).Run()
	}
	return fmt.Errorf("desktop notifications are not supported on %s; use a webhook URL", runtime.GOOS)
}

// printSchedule prints a launchd agent and a cron line that run one check
// per interval. It does not install either.
func printSchedule(in watchIn, w io.Writer) error {
	every, err := time.ParseDuration(orDefaultString(in.Every, "6h"))
	if err != nil || every < 15*time.Minute {
		return errors.New("every must be a duration of at least 15m, such as 6h")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{exe, "portfolio", "watch", "--once"}
	if in.Notify != "" {
		args = append(args, "--notify", in.Notify)
	}
	var plistArgs strings.Builder
	for _, a := range args {
		fmt.Fprintf(&plistArgs, "\n    <string>%s</string>", a)
	}
	fmt.Fprintf(w, `# macOS: save as ~/Library/LaunchAgents/com.github.justintout.go-osrs.portfolio.plist
# then run: launchctl load ~/Library/LaunchAgents/com.github.justintout.go-osrs.portfolio.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.github.justintout.go-osrs.portfolio</string>
  <key>ProgramArguments</key>
  <array>%s
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>OSRS_USER_AGENT</key>
    <string>%s</string>
  </dict>
  <key>StartInterval</key>
  <integer>%d</integer>
  <key>StandardOutPath</key>
  <string>/tmp/osrs-portfolio.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/osrs-portfolio.log</string>
</dict>
</plist>

# Linux or macOS cron (crontab -e), every %s:
%s %s >> /tmp/osrs-portfolio.log 2>&1
`, plistArgs.String(), os.Getenv("OSRS_USER_AGENT"), int(every.Seconds()), every, cronSpec(every), strings.Join(args, " "))
	return nil
}

func cronSpec(every time.Duration) string {
	switch {
	case every >= 24*time.Hour:
		return "0 9 * * *"
	case every >= time.Hour:
		return fmt.Sprintf("0 */%d * * *", int(every.Hours()))
	}
	return fmt.Sprintf("*/%d * * * *", int(every.Minutes()))
}

func orDefaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
