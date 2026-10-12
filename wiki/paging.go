package wiki

import "context"

// maxRows is the most rows the wiki returns for one Bucket query.
const maxRows = 5000

// runPaged runs q repeatedly with growing offsets until a page comes back
// short, and returns every row. Whole tables change slowly, so pages are
// cached for a day.
func runPaged[T any](ctx context.Context, c *Client, q *Query) ([]T, error) {
	var all []T
	for offset := 0; ; offset += maxRows {
		var rows []T
		if err := c.run(ctx, q.Limit(maxRows).Offset(offset), &rows, tableTTL); err != nil {
			return nil, err
		}
		all = append(all, rows...)
		if len(rows) < maxRows {
			return all, nil
		}
	}
}
