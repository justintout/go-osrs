// Package cache stores API responses on disk so repeated lookups and large
// scans do not re-download data the server has not changed.
//
// Pass a Cache to a client with its WithCache option. Entries follow the
// server's freshness headers and are revalidated with ETag or
// Last-Modified when they expire.
package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Cache is a directory of cached responses.
type Cache struct {
	dir string
}

// Entry is one cached response.
type Entry struct {
	URL          string    `json:"url"`
	Body         []byte    `json:"body"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	Stored       time.Time `json:"stored"`
	// Expires is when the entry must be revalidated before use.
	Expires time.Time `json:"expires"`
}

// Fresh reports whether e can be used without contacting the server.
func (e *Entry) Fresh(now time.Time) bool {
	return now.Before(e.Expires)
}

// New opens a cache in dir, creating it if needed.
func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	return &Cache{dir: dir}, nil
}

// Default opens the cache in the user's cache directory, such as
// ~/Library/Caches/go-osrs on macOS or ~/.cache/go-osrs on Linux.
func Default() (*Cache, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	return New(filepath.Join(base, "go-osrs"))
}

// Dir returns the cache directory.
func (c *Cache) Dir() string { return c.dir }

func (c *Cache) path(url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".json.gz")
}

// Get returns the entry for url, if any.
func (c *Cache) Get(url string) (*Entry, bool) {
	// A corrupt or truncated entry is treated as a miss and overwritten
	// by the next successful fetch.
	e, ok := c.readFile(c.path(url))
	if !ok || e.URL != url {
		return nil, false
	}
	return e, true
}

// Put stores e, replacing any entry for the same URL.
func (c *Cache) Put(e *Entry) error {
	tmp, err := os.CreateTemp(c.dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	defer os.Remove(tmp.Name())
	zw := gzip.NewWriter(tmp)
	if err := json.NewEncoder(zw).Encode(e); err != nil {
		tmp.Close()
		return fmt.Errorf("cache: %w", err)
	}
	if err := errors.Join(zw.Close(), tmp.Close()); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	// Rename is atomic, so concurrent readers never see a partial entry.
	if err := os.Rename(tmp.Name(), c.path(e.URL)); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	return nil
}

// Stats summarizes the cache's contents.
type Stats struct {
	Entries int   `json:"entries"`
	Fresh   int   `json:"fresh"`
	Bytes   int64 `json:"bytes"`
}

// Stats counts entries and their size on disk.
func (c *Cache) Stats() (Stats, error) {
	var s Stats
	now := time.Now()
	err := filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".gz" {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		s.Entries++
		s.Bytes += info.Size()
		if e, ok := c.readFile(path); ok && e.Fresh(now) {
			s.Fresh++
		}
		return nil
	})
	return s, err
}

func (c *Cache) readFile(path string) (*Entry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, false
	}
	var e Entry
	if json.NewDecoder(zr).Decode(&e) != nil {
		return nil, false
	}
	return &e, true
}

// Clear deletes every cached entry.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".gz" {
			if err := os.Remove(filepath.Join(c.dir, e.Name())); err != nil {
				return fmt.Errorf("cache: %w", err)
			}
		}
	}
	return nil
}
