package wiki

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SearchResult is one page matching a search.
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	// Edited is when the page was last edited.
	Edited time.Time `json:"edited"`
}

// Wiki namespaces for SearchQuery.
const (
	NamespaceMain   = 0
	NamespaceUpdate = 112 // game update and news posts
)

// SearchQuery configures SearchWith.
type SearchQuery struct {
	Query     string
	Namespace int
	// Sort is "relevance" (default), "last_edit_desc", or
	// "create_timestamp_desc".
	Sort  string
	Limit int
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

// Search returns up to limit main pages matching query, best match first.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return c.SearchWith(ctx, SearchQuery{Query: query, Limit: limit})
}

// SearchWith runs a search in one namespace with a chosen sort. Quote a
// phrase in Query to match it exactly.
func (c *Client) SearchWith(ctx context.Context, q SearchQuery) ([]SearchResult, error) {
	var res struct {
		Query struct {
			Search []struct {
				Title     string    `json:"title"`
				Snippet   string    `json:"snippet"`
				Timestamp time.Time `json:"timestamp"`
			} `json:"search"`
		} `json:"query"`
	}
	params := url.Values{
		"action": {"query"}, "list": {"search"}, "srsearch": {q.Query}, "srlimit": {strconv.Itoa(max(q.Limit, 1))},
		"srnamespace": {strconv.Itoa(q.Namespace)}, "srprop": {"snippet|timestamp"},
	}
	if q.Sort != "" {
		params.Set("srsort", q.Sort)
	}
	if err := c.api(ctx, params, &res); err != nil {
		return nil, err
	}
	results := make([]SearchResult, len(res.Query.Search))
	for i, r := range res.Query.Search {
		results[i] = SearchResult{Title: r.Title, Snippet: html.UnescapeString(tagRE.ReplaceAllString(r.Snippet, "")), Edited: r.Timestamp}
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

// CategoryMembers returns the titles of every page in a category, such
// as "Pets", without the "Category:" prefix.
func (c *Client) CategoryMembers(ctx context.Context, category string) ([]string, error) {
	var titles []string
	params := url.Values{"action": {"query"}, "list": {"categorymembers"}, "cmtitle": {"Category:" + category}, "cmlimit": {"500"}}
	for {
		var res struct {
			Continue struct {
				CMContinue string `json:"cmcontinue"`
			} `json:"continue"`
			Query struct {
				Members []struct {
					Title string `json:"title"`
				} `json:"categorymembers"`
			} `json:"query"`
		}
		if err := c.api(ctx, params, &res); err != nil {
			return nil, err
		}
		for _, m := range res.Query.Members {
			titles = append(titles, m.Title)
		}
		if res.Continue.CMContinue == "" {
			return titles, nil
		}
		params.Set("cmcontinue", res.Continue.CMContinue)
	}
}

// Categories returns the categories of each page, keyed by the title as
// given, without the "Category:" prefix. Redirects are followed.
func (c *Client) Categories(ctx context.Context, titles []string) (map[string][]string, error) {
	out := make(map[string][]string, len(titles))
	for start := 0; start < len(titles); start += 50 {
		batch := titles[start:min(start+50, len(titles))]
		pages, aliases, err := c.queryPages(ctx, batch, url.Values{"prop": {"categories"}, "cllimit": {"max"}})
		if err != nil {
			return nil, err
		}
		for _, t := range batch {
			for _, cat := range pages[aliases.resolve(t)].Categories {
				out[t] = append(out[t], strings.TrimPrefix(cat.Title, "Category:"))
			}
		}
	}
	return out, nil
}

// ResolveTitles maps each title to its canonical page title, following
// redirects and case normalization. Titles with no page are left out.
func (c *Client) ResolveTitles(ctx context.Context, titles []string) (map[string]string, error) {
	out := make(map[string]string, len(titles))
	for start := 0; start < len(titles); start += 50 {
		batch := titles[start:min(start+50, len(titles))]
		pages, aliases, err := c.queryPages(ctx, batch, url.Values{})
		if err != nil {
			return nil, err
		}
		for _, t := range batch {
			if p, ok := pages[aliases.resolve(t)]; ok && !p.Missing {
				out[t] = p.Title
			}
		}
	}
	return out, nil
}

type queriedPage struct {
	Title      string `json:"title"`
	Missing    bool   `json:"missing"`
	Categories []struct {
		Title string `json:"title"`
	} `json:"categories"`
}

// titleAliases maps a requested title through normalization and redirects.
type titleAliases map[string]string

func (a titleAliases) resolve(t string) string {
	for i := 0; i < 3; i++ {
		next, ok := a[t]
		if !ok {
			break
		}
		t = next
	}
	return t
}

func (c *Client) queryPages(ctx context.Context, titles []string, extra url.Values) (map[string]queriedPage, titleAliases, error) {
	params := url.Values{"action": {"query"}, "redirects": {"1"}, "titles": {strings.Join(titles, "|")}}
	for k, v := range extra {
		params[k] = v
	}
	var res struct {
		Query struct {
			Normalized []struct{ From, To string } `json:"normalized"`
			Redirects  []struct{ From, To string } `json:"redirects"`
			Pages      map[string]struct {
				Title      string  `json:"title"`
				Missing    *string `json:"missing"`
				Categories []struct {
					Title string `json:"title"`
				} `json:"categories"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.api(ctx, params, &res); err != nil {
		return nil, nil, err
	}
	aliases := titleAliases{}
	for _, n := range append(res.Query.Normalized, res.Query.Redirects...) {
		aliases[n.From] = n.To
	}
	pages := map[string]queriedPage{}
	for _, p := range res.Query.Pages {
		pages[p.Title] = queriedPage{Title: p.Title, Missing: p.Missing != nil, Categories: p.Categories}
	}
	return pages, aliases, nil
}
