package main

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/justintout/go-osrs/market"
	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

type priceHistoryIn struct {
	Item   string `json:"item" arg:"" jsonschema:"item name"`
	Step   string `json:"step,omitempty" jsonschema:"window size: 5m (default), 1h, 6h, or 24h"`
	Points int    `json:"points,omitempty" jsonschema:"windows to show, newest last, 48 by default; bursts are found across all 365 windows"`
}

type historyRow struct {
	Time       time.Time `json:"time"`
	BuyPrice   int       `json:"buy_price,omitempty"`
	BuyVolume  int       `json:"buy_volume"`
	SellPrice  int       `json:"sell_price,omitempty"`
	SellVolume int       `json:"sell_volume"`
	Update     string    `json:"update,omitempty"`
}

type priceHistoryOut struct {
	Item     string         `json:"item"`
	ID       int            `json:"id"`
	Step     string         `json:"step"`
	BuyLimit int            `json:"buy_limit"`
	Rows     []historyRow   `json:"rows"`
	Bursts   []market.Burst `json:"bursts"`
	// Updates are game updates within the series' time range.
	Updates []wiki.Update `json:"updates,omitempty"`
}

func (o priceHistoryOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (id %d), %s windows, buy limit %s\n\n", o.Item, o.ID, o.Step, num(int64(o.BuyLimit)))
	t := newTable(&b, "TIME (UTC)", "BUY PRICE", "BUY VOL", "SELL PRICE", "SELL VOL", "")
	for _, r := range o.Rows {
		t.row(r.Time.UTC().Format("Jan 02 15:04"), priceCell(r.BuyPrice), num(int64(r.BuyVolume)), priceCell(r.SellPrice), num(int64(r.SellVolume)), r.Update)
	}
	t.flush()
	if len(o.Bursts) > 0 {
		b.WriteString("\nbursts (one side far above normal volume at an off-market price):\n")
		t := newTable(&b, "TIME (UTC)", "SIDE", "PRICE", "NORMAL", "VOLUME", "VOL X", "OF BUY LIMIT")
		for _, x := range o.Bursts {
			share := "-"
			if x.BuyLimitShare > 0 {
				share = pct(x.BuyLimitShare)
			}
			t.row(x.Time.UTC().Format("Jan 02 15:04"), x.Side, num(int64(x.Price)), num(int64(x.NormalPrice)), num(int64(x.Volume)), fmt.Sprintf("%.0f", x.VolumeMultiple), share)
		}
		t.flush()
	}
	b.WriteString("\nbuy = trades at the instant-buy price; sell = trades at the instant-sell price\n")
	return b.String()
}

var priceHistoryCmd = newCommand("price-history", "show an item's price history split into instant-buy and instant-sell trades, flag single-order bursts, and mark game updates",
	func(ctx context.Context, d *deps, in priceHistoryIn) (priceHistoryOut, error) {
		if err := need("item", in.Item); err != nil {
			return priceHistoryOut{}, err
		}
		step := prices.Timestep(in.Step)
		switch step {
		case "":
			step = prices.FiveMinute
		case prices.FiveMinute, prices.OneHour, prices.SixHour, prices.TwentyFourHour:
		default:
			return priceHistoryOut{}, fmt.Errorf("step must be 5m, 1h, 6h, or 24h")
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return priceHistoryOut{}, err
		}
		ts, err := d.prices.Timeseries(ctx, item.ID, step)
		if err != nil {
			return priceHistoryOut{}, err
		}
		if len(ts.Data) == 0 {
			return priceHistoryOut{}, fmt.Errorf("no trades recorded for %s", item.Name)
		}
		out := priceHistoryOut{Item: item.Name, ID: item.ID, Step: string(step), BuyLimit: item.Limit, Bursts: market.Bursts(ts.Data, item.Limit)}

		start, end := ts.Data[0].Timestamp, ts.Data[len(ts.Data)-1].Timestamp
		updates, err := d.wiki.Updates(ctx)
		if err != nil {
			return priceHistoryOut{}, err
		}
		for _, u := range updates {
			if u.Type == "Game updates" && !u.Date.Before(start) && !u.Date.After(end) {
				out.Updates = append(out.Updates, u)
			}
		}
		rows := ts.Data[max(len(ts.Data)-orDefault(in.Points, 48), 0):]
		for _, p := range rows {
			r := historyRow{Time: p.Timestamp, BuyPrice: p.AvgHighPrice, BuyVolume: p.HighPriceVolume, SellPrice: p.AvgLowPrice, SellVolume: p.LowPriceVolume}
			for _, u := range out.Updates {
				// Mark the first window at or after the update's day.
				if !p.Timestamp.Before(u.Date.Truncate(24*time.Hour)) && p.Timestamp.Sub(u.Date.Truncate(24*time.Hour)) < stepDuration(step) {
					r.Update = "update: " + u.Title
				}
			}
			out.Rows = append(out.Rows, r)
		}
		return out, nil
	})

func stepDuration(s prices.Timestep) time.Duration {
	switch s {
	case prices.OneHour:
		return time.Hour
	case prices.SixHour:
		return 6 * time.Hour
	case prices.TwentyFourHour:
		return 24 * time.Hour
	}
	return 5 * time.Minute
}

func priceCell(p int) string {
	if p == 0 {
		return "-"
	}
	return num(int64(p))
}

type itemContextIn struct {
	Item string `json:"item" arg:"" jsonschema:"item name"`
}

type mention struct {
	Title string    `json:"title"`
	Date  time.Time `json:"date"`
}

type itemContextOut struct {
	Item      string `json:"item"`
	ID        int    `json:"id"`
	Members   bool   `json:"members"`
	BuyLimit  int    `json:"buy_limit"`
	HighAlch  int    `json:"high_alch"`
	Buy       int    `json:"buy"`
	Sell      int    `json:"sell"`
	Volume24h int    `json:"volume_24h"`
	// MadeFrom lists recipes that make the item; UsedIn lists what the
	// item is a material for. New uses are a common cause of price moves.
	MadeFrom []string `json:"made_from,omitempty"`
	UsedIn   []string `json:"used_in,omitempty"`
	Drops    []string `json:"drops,omitempty"`
	Shops    []string `json:"shops,omitempty"`
	// UpdateMentions are game update and news posts that name the item.
	UpdateMentions []mention `json:"update_mentions,omitempty"`
	// RecentEdits are wiki pages naming the item, most recently edited
	// first. A burst of edits often follows new content or a change.
	RecentEdits []mention `json:"recent_edits,omitempty"`
	// SearchLinks are community searches for discussion of the item.
	SearchLinks map[string]string `json:"search_links"`
}

func (o itemContextOut) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (id %d)  buy %s  sell %s  24h volume %s  buy limit %s  high alch %s\n",
		o.Item, o.ID, num(int64(o.Buy)), num(int64(o.Sell)), num(int64(o.Volume24h)), num(int64(o.BuyLimit)), num(int64(o.HighAlch)))
	section := func(name string, lines []string) {
		if len(lines) > 0 {
			fmt.Fprintf(&b, "\n%s:\n  %s\n", name, strings.Join(lines, "\n  "))
		}
	}
	section("made from", o.MadeFrom)
	section("used in", o.UsedIn)
	section("drops", o.Drops)
	section("shops", o.Shops)
	var ups, edits []string
	for _, m := range o.UpdateMentions {
		ups = append(ups, fmt.Sprintf("%s  %s", m.Date.Format(time.DateOnly), m.Title))
	}
	for _, m := range o.RecentEdits {
		edits = append(edits, fmt.Sprintf("%s  %s", m.Date.Format(time.DateOnly), m.Title))
	}
	section("game updates mentioning it", ups)
	section("wiki pages mentioning it, by last edit", edits)
	keys := make([]string, 0, len(o.SearchLinks))
	for k := range o.SearchLinks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var links []string
	for _, k := range keys {
		links = append(links, k+": "+o.SearchLinks[k])
	}
	section("search", links)
	return b.String()
}

var itemContextCmd = newCommand("item-context", "gather what could move an item's price: what makes it and uses it, drops and shops, its alch floor, game updates and recent wiki edits naming it, and community search links",
	func(ctx context.Context, d *deps, in itemContextIn) (itemContextOut, error) {
		if err := need("item", in.Item); err != nil {
			return itemContextOut{}, err
		}
		item, err := d.findItem(ctx, in.Item)
		if err != nil {
			return itemContextOut{}, err
		}
		spread, err := d.prices.LatestFor(ctx, item.ID)
		if err != nil {
			return itemContextOut{}, err
		}
		vols, _, err := d.prices.Volumes(ctx)
		if err != nil {
			return itemContextOut{}, err
		}
		out := itemContextOut{
			Item: item.Name, ID: item.ID, Members: item.Members, BuyLimit: item.Limit, HighAlch: item.HighAlch,
			Buy: spread.High, Sell: spread.Low, Volume24h: vols[item.ID],
		}
		q := url.QueryEscape(`"` + item.Name + `"`)
		out.SearchLinks = map[string]string{
			"reddit": "https://www.reddit.com/r/2007scape/search/?q=" + q + "&sort=new",
			"x":      "https://x.com/search?q=" + url.QueryEscape(`"`+item.Name+`" osrs`) + "&f=live",
		}

		recipes, err := d.wiki.Recipes(ctx)
		if err != nil {
			return itemContextOut{}, err
		}
		seen := map[string]bool{}
		for _, r := range recipes {
			if strings.EqualFold(r.Output.Name, item.Name) {
				var mats []string
				for _, m := range r.Materials {
					mats = append(mats, fmt.Sprintf("%gx %s", m.Quantity, m.Name))
				}
				out.MadeFrom = append(out.MadeFrom, strings.Join(mats, ", "))
			}
			for _, m := range r.Materials {
				if strings.EqualFold(m.Name, item.Name) && !seen[r.Output.Name] {
					seen[r.Output.Name] = true
					out.UsedIn = append(out.UsedIn, r.Output.Name)
				}
			}
		}
		sort.Strings(out.UsedIn)

		drops, err := d.wiki.DropSources(ctx, item.Name)
		if err != nil {
			return itemContextOut{}, err
		}
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].Probability > drops[j].Probability })
		for _, dr := range top(drops, 5) {
			out.Drops = append(out.Drops, fmt.Sprintf("%s (%s)", dr.Source, dr.Rarity))
		}
		stores, err := d.wiki.Stores(ctx)
		if err != nil {
			return itemContextOut{}, err
		}
		for _, s := range stores {
			if strings.EqualFold(s.Item, item.Name) && s.Stock != 0 && len(out.Shops) < 5 {
				out.Shops = append(out.Shops, fmt.Sprintf("%s for %d %s", s.Store, s.Price, strings.ToLower(s.Currency)))
			}
		}

		phrase := `"` + item.Name + `"`
		ups, err := d.wiki.SearchWith(ctx, wiki.SearchQuery{Query: phrase, Namespace: wiki.NamespaceUpdate, Sort: "create_timestamp_desc", Limit: 10})
		if err != nil {
			return itemContextOut{}, err
		}
		if len(ups) > 0 {
			updates, err := d.wiki.Updates(ctx)
			if err != nil {
				return itemContextOut{}, err
			}
			dates := map[string]time.Time{}
			for _, u := range updates {
				dates[u.Title] = u.Date
			}
			for _, u := range ups {
				title := strings.TrimPrefix(u.Title, "Update:")
				out.UpdateMentions = append(out.UpdateMentions, mention{Title: title, Date: dates[title]})
			}
			sort.Slice(out.UpdateMentions, func(i, j int) bool { return out.UpdateMentions[i].Date.After(out.UpdateMentions[j].Date) })
		}
		edits, err := d.wiki.SearchWith(ctx, wiki.SearchQuery{Query: phrase, Sort: "last_edit_desc", Limit: 10})
		if err != nil {
			return itemContextOut{}, err
		}
		for _, e := range edits {
			out.RecentEdits = append(out.RecentEdits, mention{Title: e.Title, Date: e.Edited})
		}
		return out, nil
	})
