package hiscores

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SkillEntry is one row of a skill leaderboard.
type SkillEntry struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	XP    int64  `json:"xp"`
}

// ActivityEntry is one row of an activity leaderboard.
type ActivityEntry struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// SkillLeaderboard returns one page of 25 rows from a skill leaderboard.
// skillID is a Skill.ID from Lookup; 0 is Overall. Pages start at 1.
func (c *Client) SkillLeaderboard(ctx context.Context, mode Mode, skillID, page int) ([]SkillEntry, error) {
	rows, err := c.leaderboard(ctx, mode, url.Values{"table": {strconv.Itoa(skillID)}, "page": {strconv.Itoa(page)}})
	if err != nil {
		return nil, err
	}
	entries := make([]SkillEntry, 0, len(rows))
	for _, r := range rows {
		// Overall rows carry an extra leading cell for the pre-Sailing rank.
		if len(r) < 4 {
			return nil, fmt.Errorf("hiscores: skill row has %d cells", len(r))
		}
		r = r[len(r)-4:]
		rank, err1 := number(r[0])
		level, err2 := number(r[2])
		xp, err3 := number(r[3])
		if err := errors.Join(err1, err2, err3); err != nil {
			return nil, err
		}
		entries = append(entries, SkillEntry{Rank: int(rank), Name: r[1], Level: int(level), XP: xp})
	}
	return entries, nil
}

// ActivityLeaderboard returns one page of 25 rows from an activity
// leaderboard. activityID is an Activity.ID from Lookup. Pages start at 1.
func (c *Client) ActivityLeaderboard(ctx context.Context, mode Mode, activityID, page int) ([]ActivityEntry, error) {
	rows, err := c.leaderboard(ctx, mode, url.Values{"category_type": {"1"}, "table": {strconv.Itoa(activityID)}, "page": {strconv.Itoa(page)}})
	if err != nil {
		return nil, err
	}
	entries := make([]ActivityEntry, 0, len(rows))
	for _, r := range rows {
		if len(r) != 3 {
			return nil, fmt.Errorf("hiscores: activity row has %d cells", len(r))
		}
		rank, err1 := number(r[0])
		score, err2 := number(r[2])
		if err := errors.Join(err1, err2); err != nil {
			return nil, err
		}
		entries = append(entries, ActivityEntry{Rank: int(rank), Name: r[1], Score: int(score)})
	}
	return entries, nil
}

var (
	rowRE  = regexp.MustCompile(`(?s)<tr class="personal-hiscores__row">(.*?)</tr>`)
	cellRE = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	tagRE  = regexp.MustCompile(`<[^>]*>`)
)

// leaderboard scrapes overall.ws. Jagex has no paged JSON leaderboard, and
// the page markup is plain enough that regexps hold up.
func (c *Client) leaderboard(ctx context.Context, mode Mode, query url.Values) ([][]string, error) {
	if p, _ := strconv.Atoi(query.Get("page")); p < 1 {
		return nil, fmt.Errorf("hiscores: page %q must be at least 1", query.Get("page"))
	}
	u, err := c.modeURL(mode, "overall.ws", query)
	if err != nil {
		return nil, err
	}
	body, err := c.t.GetBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	for _, row := range rowRE.FindAllSubmatch(body, -1) {
		var cells []string
		for _, cell := range cellRE.FindAllSubmatch(row[1], -1) {
			text := html.UnescapeString(tagRE.ReplaceAllString(string(cell[1]), ""))
			cells = append(cells, strings.TrimSpace(strings.ReplaceAll(text, " ", " ")))
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// number parses comma-grouped number text like "200,000,000".
func number(s string) (int64, error) {
	n, err := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("hiscores: parse number %q: %w", s, err)
	}
	return n, nil
}
