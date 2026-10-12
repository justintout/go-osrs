package wom

import (
	"context"
	"net/url"
	"time"
)

// Player is a player tracked by Wise Old Man.
type Player struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	// Type is the account type: regular, ironman, hardcore, ultimate, or
	// unknown.
	Type string `json:"type"`
	// Build is main, f2p, lvl3, zerker, def1, or hp10, among others.
	Build   string  `json:"build"`
	Status  string  `json:"status"`
	Country string  `json:"country"`
	Exp     int64   `json:"exp"`
	EHP     float64 `json:"ehp"`
	EHB     float64 `json:"ehb"`
	// TTM and TT200m are hours to max and to 200m XP in every skill.
	TTM           float64   `json:"ttm"`
	TT200m        float64   `json:"tt200m"`
	CombatLevel   int       `json:"combatLevel"`
	RegisteredAt  time.Time `json:"registeredAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastChangedAt time.Time `json:"lastChangedAt"`
	// LatestSnapshot is set by Player and UpdatePlayer only.
	LatestSnapshot *Snapshot `json:"latestSnapshot,omitempty"`
}

// Snapshot is a player's stats at one moment.
type Snapshot struct {
	ID        int          `json:"id"`
	PlayerID  int          `json:"playerId"`
	CreatedAt time.Time    `json:"createdAt"`
	Data      SnapshotData `json:"data"`
}

// SnapshotData maps metric names, such as "attack" or "vorkath", to values.
type SnapshotData struct {
	Skills     map[string]SkillValue    `json:"skills"`
	Bosses     map[string]BossValue     `json:"bosses"`
	Activities map[string]ActivityValue `json:"activities"`
	Computed   map[string]ComputedValue `json:"computed"`
}

// SkillValue is a skill in a snapshot. Rank and Experience are -1 when
// unranked.
type SkillValue struct {
	Metric     string  `json:"metric"`
	Experience int64   `json:"experience"`
	Rank       int     `json:"rank"`
	Level      int     `json:"level"`
	EHP        float64 `json:"ehp"`
}

// BossValue is a boss in a snapshot. Kills and Rank are -1 when unranked.
type BossValue struct {
	Metric string  `json:"metric"`
	Kills  int     `json:"kills"`
	Rank   int     `json:"rank"`
	EHB    float64 `json:"ehb"`
}

// ActivityValue is an activity in a snapshot. Score and Rank are -1 when
// unranked.
type ActivityValue struct {
	Metric string `json:"metric"`
	Score  int    `json:"score"`
	Rank   int    `json:"rank"`
}

// ComputedValue is a derived metric, such as ehp or ehb.
type ComputedValue struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Rank   int     `json:"rank"`
}

// Player returns a tracked player with their latest snapshot. It returns
// an error matching osrs.ErrNotFound if Wise Old Man does not track them.
func (c *Client) Player(ctx context.Context, username string) (*Player, error) {
	return get[*Player](ctx, c, playerPath(username), nil)
}

// UpdatePlayer asks Wise Old Man to fetch the player's current hiscores,
// starting tracking if needed, and returns the updated player.
func (c *Client) UpdatePlayer(ctx context.Context, username string) (*Player, error) {
	var p Player
	if err := c.t.PostJSON(ctx, c.baseURL+playerPath(username), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Snapshots returns the player's snapshots within period, newest first.
func (c *Client) Snapshots(ctx context.Context, username string, period Period) ([]Snapshot, error) {
	return get[[]Snapshot](ctx, c, playerPath(username, "snapshots"), url.Values{"period": {string(period)}})
}

// Achievement is a milestone a player reached.
type Achievement struct {
	PlayerID  int       `json:"playerId"`
	Name      string    `json:"name"`
	Metric    string    `json:"metric"`
	Measure   string    `json:"measure"`
	Threshold int64     `json:"threshold"`
	CreatedAt time.Time `json:"createdAt"`
}

// Achievements returns the milestones a player has reached.
func (c *Client) Achievements(ctx context.Context, username string) ([]Achievement, error) {
	return get[[]Achievement](ctx, c, playerPath(username, "achievements"), nil)
}

// BossRates returns the kills per hour Wise Old Man assumes for each boss
// when computing EHB, keyed by boss metric such as "vorkath". accountType
// is "main" or "ironman".
func (c *Client) BossRates(ctx context.Context, accountType string) (map[string]float64, error) {
	rows, err := get[[]struct {
		Boss string  `json:"boss"`
		Rate float64 `json:"rate"`
	}](ctx, c, "/efficiency/rates", url.Values{"metric": {"ehb"}, "type": {accountType}})
	if err != nil {
		return nil, err
	}
	rates := make(map[string]float64, len(rows))
	for _, r := range rows {
		rates[r.Boss] = r.Rate
	}
	return rates, nil
}
