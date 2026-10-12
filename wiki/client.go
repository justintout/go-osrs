// Package wiki queries the OSRS Wiki: structured Bucket data for items,
// monsters, drops, and equipment, plus page search and summaries.
// https://oldschool.runescape.wiki/w/RuneScape:Bucket
package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/justintout/go-osrs/internal/transport"
)

const defaultAPIURL = "https://oldschool.runescape.wiki/api.php"

// Client queries the OSRS Wiki API.
type Client struct {
	t      transport.Client
	apiURL string
}

// NewClient creates a Client. The Wiki team requires a descriptive user
// agent.
func NewClient(userAgent string, options ...Option) *Client {
	c := &Client{
		t:      transport.Client{HTTP: http.DefaultClient, UserAgent: userAgent},
		apiURL: defaultAPIURL,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.t.HTTP = httpClient }
}

// WithAPIURL sets the api.php URL, for proxies and tests.
func WithAPIURL(url string) Option {
	return func(c *Client) { c.apiURL = url }
}

func (c *Client) api(ctx context.Context, params url.Values, v any) error {
	params.Set("format", "json")
	return c.t.GetJSON(ctx, c.apiURL+"?"+params.Encode(), v)
}

// Query builds a Bucket query. Bucket schemas are listed at
// https://oldschool.runescape.wiki/w/Special:AllPages?namespace=9592.
type Query struct {
	bucket string
	fields []string
	wheres []string
	limit  int
	offset int
}

// Bucket starts a query on the named bucket, such as "infobox_item".
func Bucket(name string) *Query {
	return &Query{bucket: name}
}

// Select adds fields to return. Every bucket also has page_name.
func (q *Query) Select(fields ...string) *Query {
	q.fields = append(q.fields, fields...)
	return q
}

// Where keeps rows whose field equals value. Strings, numbers, and bools
// are supported.
func (q *Query) Where(field string, value any) *Query {
	q.wheres = append(q.wheres, fmt.Sprintf("where(%s,%s)", quote(field), literal(value)))
	return q
}

// Limit caps the number of rows. The wiki's maximum is 5000.
func (q *Query) Limit(n int) *Query {
	q.limit = n
	return q
}

// Offset skips the first n rows, for paging.
func (q *Query) Offset(n int) *Query {
	q.offset = n
	return q
}

// String returns the query in Bucket syntax.
func (q *Query) String() string {
	var b strings.Builder
	b.WriteString("bucket(" + quote(q.bucket) + ")")
	if len(q.fields) > 0 {
		quoted := make([]string, len(q.fields))
		for i, f := range q.fields {
			quoted[i] = quote(f)
		}
		b.WriteString(".select(" + strings.Join(quoted, ",") + ")")
	}
	for _, w := range q.wheres {
		b.WriteString("." + w)
	}
	if q.limit > 0 {
		fmt.Fprintf(&b, ".limit(%d)", q.limit)
	}
	if q.offset > 0 {
		fmt.Fprintf(&b, ".offset(%d)", q.offset)
	}
	b.WriteString(".run()")
	return b.String()
}

// Run executes q and decodes its rows into rows, which should point to a
// slice of structs with json tags matching the selected fields. Use Flag
// for BOOLEAN fields.
func (c *Client) Run(ctx context.Context, q *Query, rows any) error {
	var res struct {
		Error  string          `json:"error"`
		Bucket json.RawMessage `json:"bucket"`
	}
	if err := c.api(ctx, url.Values{"action": {"bucket"}, "query": {q.String()}}, &res); err != nil {
		return err
	}
	if res.Error != "" {
		return fmt.Errorf("wiki: bucket query %s: %s", q, res.Error)
	}
	if err := json.Unmarshal(res.Bucket, rows); err != nil {
		return fmt.Errorf("wiki: decode bucket rows: %w", err)
	}
	return nil
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func literal(v any) string {
	if s, ok := v.(string); ok {
		return quote(s)
	}
	return fmt.Sprint(v)
}
