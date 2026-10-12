package wom

import (
	"context"
	"net/url"
	"time"
)

// Delta is a value's change over a time range.
type Delta[T int | int64 | float64] struct {
	Gained T `json:"gained"`
	Start  T `json:"start"`
	End    T `json:"end"`
}

// Gains is a player's progress over a time range.
type Gains struct {
	StartsAt time.Time `json:"startsAt"`
	EndsAt   time.Time `json:"endsAt"`
	Data     GainsData `json:"data"`
}

// GainsData maps metric names, such as "attack" or "vorkath", to deltas.
type GainsData struct {
	Skills     map[string]SkillGain    `json:"skills"`
	Bosses     map[string]BossGain     `json:"bosses"`
	Activities map[string]ActivityGain `json:"activities"`
	Computed   map[string]ComputedGain `json:"computed"`
}

// SkillGain is a skill's change over a time range.
type SkillGain struct {
	Metric     string         `json:"metric"`
	Experience Delta[int64]   `json:"experience"`
	Level      Delta[int]     `json:"level"`
	Rank       Delta[int]     `json:"rank"`
	EHP        Delta[float64] `json:"ehp"`
}

// BossGain is a boss's change over a time range.
type BossGain struct {
	Metric string         `json:"metric"`
	Kills  Delta[int]     `json:"kills"`
	Rank   Delta[int]     `json:"rank"`
	EHB    Delta[float64] `json:"ehb"`
}

// ActivityGain is an activity's change over a time range.
type ActivityGain struct {
	Metric string     `json:"metric"`
	Score  Delta[int] `json:"score"`
	Rank   Delta[int] `json:"rank"`
}

// ComputedGain is a derived metric's change over a time range.
type ComputedGain struct {
	Metric string         `json:"metric"`
	Value  Delta[float64] `json:"value"`
	Rank   Delta[int]     `json:"rank"`
}

// Gains returns a player's progress over period.
func (c *Client) Gains(ctx context.Context, username string, period Period) (*Gains, error) {
	return get[*Gains](ctx, c, playerPath(username, "gained"), url.Values{"period": {string(period)}})
}

// GainsBetween returns a player's progress between start and end.
func (c *Client) GainsBetween(ctx context.Context, username string, start, end time.Time) (*Gains, error) {
	q := url.Values{"startDate": {start.UTC().Format(time.RFC3339)}, "endDate": {end.UTC().Format(time.RFC3339)}}
	return get[*Gains](ctx, c, playerPath(username, "gained"), q)
}
