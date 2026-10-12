package wiki

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strconv"
)

// SearchResult is one page matching a search.
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

// Search returns up to limit pages matching query, best match first.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	var res struct {
		Query struct {
			Search []SearchResult `json:"search"`
		} `json:"query"`
	}
	params := url.Values{"action": {"query"}, "list": {"search"}, "srsearch": {query}, "srlimit": {strconv.Itoa(limit)}}
	if err := c.api(ctx, params, &res); err != nil {
		return nil, err
	}
	results := res.Query.Search
	for i := range results {
		results[i].Snippet = html.UnescapeString(tagRE.ReplaceAllString(results[i].Snippet, ""))
	}
	return results, nil
}

// Summary returns the plain-text introduction of a page, following
// redirects. It returns "" for a missing page.
func (c *Client) Summary(ctx context.Context, title string) (string, error) {
	var res struct {
		Query struct {
			Pages map[string]struct {
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}
	params := url.Values{"action": {"query"}, "prop": {"extracts"}, "exintro": {"1"}, "explaintext": {"1"}, "redirects": {"1"}, "titles": {title}}
	if err := c.api(ctx, params, &res); err != nil {
		return "", err
	}
	for _, p := range res.Query.Pages {
		return p.Extract, nil
	}
	return "", nil
}
