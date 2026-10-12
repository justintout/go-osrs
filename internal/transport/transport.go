// Package transport sends the HTTP requests shared by the API clients:
// it sets the User-Agent, spaces requests per host, merges identical
// concurrent requests, caches responses, and maps errors.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/cache"
)

// Client sends requests with a required User-Agent.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	// Header holds extra headers sent with every request, such as API keys.
	Header http.Header
	// Cache stores GET responses when set.
	Cache *cache.Cache
	// MinInterval is the least time between requests to one host.
	MinInterval time.Duration

	mu      sync.Mutex
	next    map[string]time.Time
	flights map[string]*flight
}

// New returns a Client.
func New(httpClient *http.Client, userAgent string, minInterval time.Duration) *Client {
	return &Client{
		HTTP: httpClient, UserAgent: userAgent, Header: http.Header{}, MinInterval: minInterval,
		next: map[string]time.Time{}, flights: map[string]*flight{},
	}
}

// Option changes how one request is cached.
type Option func(*options)

type options struct {
	minTTL time.Duration
}

// MinTTL keeps the response cached for at least d, even if the server
// says it expires sooner. Use it for data that never changes, such as
// price averages for a window that has closed.
func MinTTL(d time.Duration) Option {
	return func(o *options) { o.minTTL = d }
}

// GetJSON fetches url and decodes the JSON body into v.
func (c *Client) GetJSON(ctx context.Context, url string, v any, opts ...Option) error {
	b, err := c.GetBytes(ctx, url, opts...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("osrs: decode %s: %w", url, err)
	}
	return nil
}

// GetBytes fetches url and returns the body, from the cache when fresh.
func (c *Client) GetBytes(ctx context.Context, url string, opts ...Option) ([]byte, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return c.once(url, func() ([]byte, error) { return c.cachedGet(ctx, url, o) })
}

// PostJSON sends an empty POST to url and decodes the JSON body into v.
// POSTs are never cached.
func (c *Client) PostJSON(ctx context.Context, url string, v any) error {
	res, err := c.send(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := statusError(res, url); err != nil {
		return err
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("osrs: decode %s: %w", url, err)
	}
	return nil
}

func (c *Client) cachedGet(ctx context.Context, url string, o options) ([]byte, error) {
	var stale *cache.Entry
	if c.Cache != nil {
		if e, ok := c.Cache.Get(url); ok {
			if e.Fresh(time.Now()) {
				return e.Body, nil
			}
			stale = e
		}
	}
	res, err := c.send(ctx, http.MethodGet, url, stale)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	now := time.Now()
	if res.StatusCode == http.StatusNotModified && stale != nil {
		stale.Stored, stale.Expires = now, expiry(res.Header, now, o.minTTL)
		c.store(stale)
		return stale.Body, nil
	}
	if err := statusError(res, url); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("osrs: read %s: %w", url, err)
	}
	e := &cache.Entry{
		URL: url, Body: body, ETag: res.Header.Get("ETag"), LastModified: res.Header.Get("Last-Modified"),
		Stored: now, Expires: expiry(res.Header, now, o.minTTL),
	}
	// Store what is fresh for a while or can be revalidated. Responses
	// marked no-store, or expired with no validator, are not reusable.
	reusable := e.Fresh(now) || e.ETag != "" || e.LastModified != ""
	if c.Cache != nil && reusable && !hasDirective(res.Header, "no-store") {
		c.store(e)
	}
	return body, nil
}

// store writes to the cache. A failed write only costs a future refetch,
// so the response is still returned.
func (c *Client) store(e *cache.Entry) {
	_ = c.Cache.Put(e)
}

// send makes one request, spacing requests per host and waiting out a
// short Retry-After on 429. A stale entry's validators make it
// conditional.
func (c *Client) send(ctx context.Context, method, rawURL string, stale *cache.Entry) (*http.Response, error) {
	// The OSRS Wiki and Jagex block generic user agents.
	if c.UserAgent == "" {
		return nil, errors.New("osrs: a descriptive user agent is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("osrs: %w", err)
	}
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx, u.Host); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("osrs: build request: %w", err)
		}
		for k, v := range c.Header {
			req.Header[k] = v
		}
		req.Header.Set("User-Agent", c.UserAgent)
		if stale != nil {
			if stale.ETag != "" {
				req.Header.Set("If-None-Match", stale.ETag)
			}
			if stale.LastModified != "" {
				req.Header.Set("If-Modified-Since", stale.LastModified)
			}
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("osrs: %w", err)
		}
		delay, retry := retryAfter(res)
		if res.StatusCode != http.StatusTooManyRequests || !retry || attempt >= 2 {
			return res, nil
		}
		res.Body.Close()
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
}

// wait blocks until MinInterval has passed since the last request to host.
func (c *Client) wait(ctx context.Context, host string) error {
	if c.MinInterval <= 0 {
		return nil
	}
	c.mu.Lock()
	now := time.Now()
	at := c.next[host]
	if at.Before(now) {
		at = now
	}
	c.next[host] = at.Add(c.MinInterval)
	c.mu.Unlock()
	return sleep(ctx, time.Until(at))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryAfter reports a 429's Retry-After delay when it is short enough to
// wait out.
func retryAfter(res *http.Response) (time.Duration, bool) {
	secs, err := strconv.Atoi(res.Header.Get("Retry-After"))
	if err != nil || secs < 0 || secs > 30 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

type flight struct {
	done chan struct{}
	body []byte
	err  error
}

// once runs fn for url, sharing the result with identical requests that
// arrive while it runs.
func (c *Client) once(url string, fn func() ([]byte, error)) ([]byte, error) {
	c.mu.Lock()
	if f, ok := c.flights[url]; ok {
		c.mu.Unlock()
		<-f.done
		return f.body, f.err
	}
	f := &flight{done: make(chan struct{})}
	c.flights[url] = f
	c.mu.Unlock()

	f.body, f.err = fn()
	c.mu.Lock()
	delete(c.flights, url)
	c.mu.Unlock()
	close(f.done)
	return f.body, f.err
}

func statusError(res *http.Response, url string) error {
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return &osrs.StatusError{StatusCode: res.StatusCode, URL: url}
	}
	return nil
}

// expiry is when a response stops being fresh: from Cache-Control max-age
// less the response's Age, else Expires, and at least now+minTTL.
func expiry(h http.Header, now time.Time, minTTL time.Duration) time.Time {
	exp := now
	switch {
	case hasDirective(h, "no-cache"), hasDirective(h, "no-store"):
	case directive(h, "max-age") != "":
		maxAge, _ := strconv.Atoi(directive(h, "max-age"))
		age, _ := strconv.Atoi(h.Get("Age"))
		exp = now.Add(time.Duration(maxAge-age) * time.Second)
	case h.Get("Expires") != "":
		if t, err := http.ParseTime(h.Get("Expires")); err == nil {
			exp = t
		}
	}
	if floor := now.Add(minTTL); exp.Before(floor) {
		exp = floor
	}
	return exp
}

func hasDirective(h http.Header, name string) bool {
	for _, d := range strings.Split(h.Get("Cache-Control"), ",") {
		if strings.EqualFold(strings.TrimSpace(d), name) {
			return true
		}
	}
	return false
}

func directive(h http.Header, name string) string {
	for _, d := range strings.Split(h.Get("Cache-Control"), ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(d), "=")
		if ok && strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}
