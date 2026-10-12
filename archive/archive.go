// Package archive keeps Grand Exchange price history in SQLite. It fills
// in only the windows a caller asks for and does not already have, so
// history builds up from use instead of a bulk download.
package archive

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/justintout/go-osrs/prices"
)

const schema = `
CREATE TABLE IF NOT EXISTS windows (
	step  TEXT    NOT NULL,
	start INTEGER NOT NULL,
	PRIMARY KEY (step, start)
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS averages (
	step     TEXT    NOT NULL,
	item     INTEGER NOT NULL,
	start    INTEGER NOT NULL,
	avg_high INTEGER NOT NULL,
	high_vol INTEGER NOT NULL,
	avg_low  INTEGER NOT NULL,
	low_vol  INTEGER NOT NULL,
	PRIMARY KEY (step, item, start)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS averages_window ON averages (step, start);`

// Archive is a SQLite database of price averages by item and window.
type Archive struct {
	mu     sync.Mutex
	conn   *sqlite.Conn
	prices *prices.Client
}

// Open opens or creates the archive at path. p fetches missing windows.
func Open(path string, p *prices.Client) (*Archive, error) {
	conn, err := sqlite.OpenConn(path)
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	if err := sqlitex.ExecuteScript(conn, "PRAGMA journal_mode=WAL;"+schema, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("archive: %w", err)
	}
	return &Archive{conn: conn, prices: p}, nil
}

// Close closes the database.
func (a *Archive) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn.Close()
}

// Starts returns the window starts of step that begin in [from, to) and
// have closed, oldest first.
func Starts(step prices.Timestep, from, to time.Time) ([]time.Time, error) {
	d, err := step.Duration()
	if err != nil {
		return nil, err
	}
	if latest := time.Now().Add(-d).Truncate(d); to.After(latest.Add(d)) {
		to = latest.Add(d)
	}
	var out []time.Time
	for t := from.Truncate(d); t.Before(to); t = t.Add(d) {
		if !t.Before(from) {
			out = append(out, t.UTC())
		}
	}
	return out, nil
}

// Ensure fetches each closed window of step in [from, to) that the archive
// lacks and reports how many it fetched. A window with no trades is still
// recorded, so it is not fetched again.
func (a *Archive) Ensure(ctx context.Context, step prices.Timestep, from, to time.Time) (int, error) {
	starts, err := Starts(step, from, to)
	if err != nil {
		return 0, err
	}
	have, err := a.covered(step, from, to)
	if err != nil {
		return 0, err
	}
	fetched := 0
	for _, s := range starts {
		if have[s.Unix()] {
			continue
		}
		avgs, err := a.prices.AveragesAt(ctx, step, s)
		if err != nil {
			return fetched, err
		}
		if err := a.insert(step, s, avgs); err != nil {
			return fetched, err
		}
		fetched++
	}
	return fetched, nil
}

func (a *Archive) covered(step prices.Timestep, from, to time.Time) (map[int64]bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	have := map[int64]bool{}
	err := sqlitex.Execute(a.conn, "SELECT start FROM windows WHERE step = ? AND start >= ? AND start < ?", &sqlitex.ExecOptions{
		Args: []any{string(step), from.Unix(), to.Unix()},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			have[stmt.ColumnInt64(0)] = true
			return nil
		},
	})
	return have, err
}

func (a *Archive) insert(step prices.Timestep, start time.Time, avgs prices.Averages) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	defer sqlitex.Save(a.conn)(&err)
	for id, v := range avgs {
		err := sqlitex.Execute(a.conn, "INSERT OR REPLACE INTO averages VALUES (?, ?, ?, ?, ?, ?, ?)", &sqlitex.ExecOptions{
			Args: []any{string(step), id, start.Unix(), v.AvgHighPrice, v.HighPriceVolume, v.AvgLowPrice, v.LowPriceVolume},
		})
		if err != nil {
			return fmt.Errorf("archive: %w", err)
		}
	}
	return sqlitex.Execute(a.conn, "INSERT OR REPLACE INTO windows VALUES (?, ?)", &sqlitex.ExecOptions{Args: []any{string(step), start.Unix()}})
}

// Series returns every archived item's windows in [from, to), keyed by
// item ID, oldest first. Call Ensure first to fill gaps.
func (a *Archive) Series(step prices.Timestep, from, to time.Time) (map[int][]prices.TimeseriesDatapoint, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[int][]prices.TimeseriesDatapoint{}
	err := sqlitex.Execute(a.conn,
		"SELECT item, start, avg_high, high_vol, avg_low, low_vol FROM averages WHERE step = ? AND start >= ? AND start < ? ORDER BY item, start",
		&sqlitex.ExecOptions{
			Args: []any{string(step), from.Unix(), to.Unix()},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				id := int(stmt.ColumnInt64(0))
				out[id] = append(out[id], prices.TimeseriesDatapoint{
					Timestamp:    time.Unix(stmt.ColumnInt64(1), 0).UTC(),
					AvgHighPrice: int(stmt.ColumnInt64(2)), HighPriceVolume: int(stmt.ColumnInt64(3)),
					AvgLowPrice: int(stmt.ColumnInt64(4)), LowPriceVolume: int(stmt.ColumnInt64(5)),
				})
				return nil
			},
		})
	return out, err
}
