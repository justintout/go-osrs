package portfolio

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/runelite"
	"github.com/justintout/go-osrs/wiki"
)

// kinds returns alert keys without the position ID, such as "exit|open".
func kinds(r Report) map[string]bool {
	k := map[string]bool{}
	for _, a := range r.Alerts {
		k[strings.TrimPrefix(a.Key, r.Position.ID+"|")] = true
	}
	return k
}

func TestEvaluate(t *testing.T) {
	opened := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	p := Position{ID: "zenyte", Item: "Zenyte shard", ItemID: 19529, Quantity: 1, Filled: 1, Price: 15_000_000,
		Opened: opened, Target: 19_000_000, ExitFrom: opened.AddDate(0, 3, 0), ExitBy: opened.AddDate(0, 4, 16)}

	r := Evaluate(p, prices.Spread{High: 16_400_000, Low: 16_300_000}, nil, opened.AddDate(0, 0, 10))
	if len(r.Alerts) != 0 || r.Cost != 15_000_000 || r.Value != 16_300_000-326_000 {
		t.Errorf("quiet check = %+v", r)
	}

	upd := []wiki.Update{{Title: "Gorilla changes", Date: opened.AddDate(0, 1, 0)}, {Title: "Old news", Date: opened.AddDate(-1, 0, 0)}}
	r = Evaluate(p, prices.Spread{High: 19_100_000, Low: 12_000_000}, upd, opened.AddDate(0, 3, 5))
	got := kinds(r)
	for _, want := range []string{"target|19000000", "drawdown|15", "exit|open", "news|Gorilla changes"} {
		if !got[want] {
			t.Errorf("missing alert %s in %v", want, got)
		}
	}
	if got["news|Old news"] {
		t.Error("alerted on an update from before the position opened")
	}

	offer := Position{ID: "dip", Item: "Zenyte shard", ItemID: 19529, Quantity: 1, Price: 14_000_000, Opened: opened}
	if k := kinds(Evaluate(offer, prices.Spread{Low: 16_400_000}, nil, opened.AddDate(0, 0, 5))); !k["offer|stale"] {
		t.Errorf("stale offer not flagged: %v", k)
	}
	if k := kinds(Evaluate(offer, prices.Spread{Low: 13_500_000}, nil, opened.AddDate(0, 0, 5))); !k["offer|fillable"] {
		t.Errorf("fillable offer not flagged: %v", k)
	}
}

func TestBook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p", "portfolio.json")
	b, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := b.Add(Position{Item: "Zenyte shard", Quantity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if b.Positions[1].ID != "zenyte-2" {
		t.Errorf("ids = %s, %s", b.Positions[0].ID, b.Positions[1].ID)
	}
	b.Sent["k"] = time.Now()
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	b2, err := Load(path)
	if err != nil || len(b2.Positions) != 2 || b2.Sent["k"].IsZero() {
		t.Errorf("reloaded = %+v, %v", b2, err)
	}
}

func TestSync(t *testing.T) {
	b := &Book{}
	for _, p := range []Position{
		{ID: "zenyte", ItemID: 19529, Quantity: 1, Price: 15_000_000},
		{ID: "zenyte-dip", ItemID: 19529, Quantity: 1, Price: 14_000_000},
		{ID: "antifire", ItemID: 22209, Quantity: 500, Price: 20_000, Filled: 80},
		{ID: "held", ItemID: 1, Quantity: 10, Price: 100, Filled: 10},
	} {
		if _, err := b.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	offers := []runelite.Offer{
		{ItemID: 19529, Price: 14_000_000, TotalQuantity: 1, QuantitySold: 1, Spent: 13_900_000, State: "BOUGHT"},
		{ItemID: 22209, Price: 20_000, TotalQuantity: 500, QuantitySold: 50, Spent: 1_000_000, State: "BUYING"},
		{ItemID: 1, Price: 150, TotalQuantity: 10, QuantitySold: 10, Spent: 1_500, State: "SOLD"},
		{ItemID: 99, Price: 5, TotalQuantity: 3, State: "BUYING"},
	}
	changes := Sync(b, offers, time.Now())
	dip, _ := b.Find("zenyte-dip")
	zen, _ := b.Find("zenyte")
	anti, _ := b.Find("antifire")
	held, _ := b.Find("held")
	if dip.Filled != 1 || dip.Price != 13_900_000 || zen.Filled != 0 {
		t.Errorf("zenyte = %+v, dip = %+v", zen, dip)
	}
	if anti.Filled != 80 {
		t.Errorf("sync lowered a newer manual fill: %+v", anti)
	}
	if held.Open() || held.ClosedPrice != 150 {
		t.Errorf("held = %+v", held)
	}
	if len(changes) != 3 || changes[2].Position != "" {
		t.Errorf("changes = %+v", changes)
	}
}
