package wom

import (
	"context"
	"net/url"
	"time"
)

// Group is a Wise Old Man group, usually a clan.
type Group struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	ClanChat    string       `json:"clanChat"`
	Description string       `json:"description"`
	Homeworld   int          `json:"homeworld"`
	Verified    bool         `json:"verified"`
	MemberCount int          `json:"memberCount"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Memberships []Membership `json:"memberships"`
}

// Membership is a player's place in a group.
type Membership struct {
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	Player    Player    `json:"player"`
}

// Group returns a group and its members.
func (c *Client) Group(ctx context.Context, id int) (*Group, error) {
	return get[*Group](ctx, c, idPath("groups", id, ""), nil)
}

// MemberGain is one member's progress in a metric.
type MemberGain struct {
	Player    Player       `json:"player"`
	StartDate time.Time    `json:"startDate"`
	EndDate   time.Time    `json:"endDate"`
	Data      Delta[int64] `json:"data"`
}

// GroupGains returns each member's progress in metric over period, most
// gained first. metric is a Wise Old Man metric such as "overall",
// "attack", or "vorkath".
func (c *Client) GroupGains(ctx context.Context, id int, metric string, period Period) ([]MemberGain, error) {
	q := url.Values{"metric": {metric}, "period": {string(period)}}
	return get[[]MemberGain](ctx, c, idPath("groups", id, "/gained"), q)
}

// MemberStanding is one member's current value in a metric. Only the
// fields for the metric's type are set: Level and Experience for skills,
// Kills for bosses, Score for activities.
type MemberStanding struct {
	Player Player `json:"player"`
	Data   struct {
		Type       string `json:"type"`
		Rank       int    `json:"rank"`
		Level      int    `json:"level,omitempty"`
		Experience int64  `json:"experience,omitempty"`
		Kills      int    `json:"kills,omitempty"`
		Score      int    `json:"score,omitempty"`
	} `json:"data"`
}

// GroupHiscores returns members ranked by their current value in metric.
func (c *Client) GroupHiscores(ctx context.Context, id int, metric string) ([]MemberStanding, error) {
	q := url.Values{"metric": {metric}}
	return get[[]MemberStanding](ctx, c, idPath("groups", id, "/hiscores"), q)
}

// Competition is a time-boxed contest in one metric.
type Competition struct {
	ID               int             `json:"id"`
	Title            string          `json:"title"`
	Metric           string          `json:"metric"`
	Type             string          `json:"type"`
	StartsAt         time.Time       `json:"startsAt"`
	EndsAt           time.Time       `json:"endsAt"`
	GroupID          int             `json:"groupId"`
	ParticipantCount int             `json:"participantCount"`
	Participations   []Participation `json:"participations"`
}

// Participation is one player's progress in a competition.
type Participation struct {
	Player   Player       `json:"player"`
	TeamName string       `json:"teamName"`
	Progress Delta[int64] `json:"progress"`
	Levels   Delta[int]   `json:"levels"`
}

// Competition returns a competition with its participants' progress.
func (c *Client) Competition(ctx context.Context, id int) (*Competition, error) {
	return get[*Competition](ctx, c, idPath("competitions", id, ""), nil)
}
