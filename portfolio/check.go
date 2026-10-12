package portfolio

import (
	"fmt"
	"time"

	"github.com/justintout/go-osrs/prices"
	"github.com/justintout/go-osrs/wiki"
)

// Alert is something about a position that needs the owner's attention.
type Alert struct {
	// Kind is target, exit, drawdown, offer, or news.
	Kind    string `json:"kind"`
	Message string `json:"message"`
	// Key identifies the alert across checks, so it is sent once.
	Key string `json:"key"`
}

// Report is one position's state at a check.
type Report struct {
	Position Position `json:"position"`
	// Buy and Sell are the latest instant-buy and instant-sell prices.
	Buy  int `json:"buy"`
	Sell int `json:"sell"`
	// Cost, Value, and PnL cover filled units. Value is what selling them
	// instantly would bring after tax.
	Cost   int     `json:"cost"`
	Value  int     `json:"value"`
	PnL    int     `json:"pnl"`
	PnLPct float64 `json:"pnl_pct"`
	Alerts []Alert `json:"alerts"`
}

// Evaluate checks an open position against the latest prices and the game
// updates that mention its item.
func Evaluate(p Position, latest prices.Spread, updates []wiki.Update, now time.Time) Report {
	r := Report{Position: p, Buy: latest.High, Sell: latest.Low}
	alert := func(kind, detail, msg string, args ...any) {
		r.Alerts = append(r.Alerts, Alert{Kind: kind, Message: fmt.Sprintf(msg, args...), Key: p.ID + "|" + kind + "|" + detail})
	}

	if p.Filled > 0 {
		r.Cost = p.Filled * p.Price
		r.Value = p.Filled * prices.AfterTax(p.ItemID, latest.Low)
		r.PnL = r.Value - r.Cost
		r.PnLPct = float64(r.PnL) / float64(r.Cost)
		limit := p.MaxDrawdown
		if limit <= 0 {
			limit = 0.15
		}
		if r.PnLPct <= -limit {
			alert("drawdown", fmt.Sprintf("%.0f", limit*100), "%s is down %.1f%% after tax; review the plan", p.Item, -r.PnLPct*100)
		}
		if p.Target > 0 && latest.High >= p.Target {
			alert("target", fmt.Sprint(p.Target), "%s buyers are paying %s, at or above the %s target; a sell offer there should fill", p.Item, gp(latest.High), gp(p.Target))
		}
	}

	if p.Filled < p.Quantity && latest.Low > 0 {
		gap := float64(latest.Low)/float64(p.Price) - 1
		switch {
		case latest.Low <= p.Price:
			alert("offer", "fillable", "%s sellers are at %s, at or below your %s offer; check the GE for fills", p.Item, gp(latest.Low), gp(p.Price))
		case gap > 0.05 && now.Sub(p.Opened) > 72*time.Hour:
			alert("offer", "stale", "%s offer at %s is %.0f%% below the market (%s) after %d days; it fills only on a dip", p.Item, gp(p.Price), gap*100, gp(latest.Low), int(now.Sub(p.Opened).Hours()/24))
		}
	}

	switch {
	case !p.ExitBy.IsZero() && now.After(p.ExitBy) && p.Filled > 0:
		alert("exit", "passed", "%s is past its %s exit date; sell", p.Item, p.ExitBy.Format(time.DateOnly))
	case !p.ExitFrom.IsZero() && !now.Before(p.ExitFrom) && p.Filled > 0:
		alert("exit", "open", "%s exit window is open until %s", p.Item, p.ExitBy.Format(time.DateOnly))
	}

	for _, u := range updates {
		if u.Date.After(p.Opened.Add(-24 * time.Hour)) {
			alert("news", u.Title, "game update %q (%s) mentions %s; check whether it changes the plan", u.Title, u.Date.Format(time.DateOnly), p.Item)
		}
	}
	return r
}

// gp formats coins with thousands separators.
func gp(n int) string {
	s := fmt.Sprint(n)
	var b []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, s[i])
	}
	return string(b)
}
