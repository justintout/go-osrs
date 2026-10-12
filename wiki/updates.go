package wiki

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Update is a game update or patch note post.
type Update struct {
	Title string    `json:"title"`
	Type  string    `json:"type"`
	Date  time.Time `json:"date"`
}

// Updates returns every game update the wiki records, newest first.
func (c *Client) Updates(ctx context.Context) ([]Update, error) {
	rows, err := runPaged[struct {
		Page string `json:"page_name"`
		Type string `json:"type"`
		Date int64  `json:"date"`
	}](ctx, c, Bucket("update").Select("page_name", "type", "date"))
	if err != nil {
		return nil, err
	}
	updates := make([]Update, len(rows))
	for i, r := range rows {
		updates[i] = Update{Title: strings.TrimPrefix(r.Page, "Update:"), Type: r.Type, Date: time.Unix(r.Date, 0).UTC()}
	}
	sort.Slice(updates, func(i, j int) bool { return updates[i].Date.After(updates[j].Date) })
	return updates, nil
}
