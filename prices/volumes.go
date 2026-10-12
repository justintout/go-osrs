package prices

import (
	"context"
	"time"
)

// Volumes returns the number of each item traded on the Grand Exchange in
// the last 24 hours, keyed by item ID, and the time of the snapshot.
func (c *Client) Volumes(ctx context.Context) (map[int]int, time.Time, error) {
	var v struct {
		Timestamp int64       `json:"timestamp"`
		Data      map[int]int `json:"data"`
	}
	if err := c.t.GetJSON(ctx, c.baseURL+"/volumes", &v); err != nil {
		return nil, time.Time{}, err
	}
	return v.Data, time.Unix(v.Timestamp, 0), nil
}
