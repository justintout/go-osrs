package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/cache"
)

func TestGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte(`{"n": 1}`))
		case "/limited":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New(srv.Client(), "test-agent", 0)
	ctx := context.Background()

	var v struct{ N int }
	if err := c.GetJSON(ctx, srv.URL+"/ok", &v); err != nil || v.N != 1 {
		t.Fatalf("GetJSON = %v, %+v", err, v)
	}
	if err := c.GetJSON(ctx, srv.URL+"/missing", &v); !errors.Is(err, osrs.ErrNotFound) {
		t.Errorf("missing: err = %v, want ErrNotFound", err)
	}
	if err := c.GetJSON(ctx, srv.URL+"/limited", &v); !errors.Is(err, osrs.ErrRateLimited) {
		t.Errorf("limited: err = %v, want ErrRateLimited", err)
	}
	c.UserAgent = ""
	if err := c.GetJSON(ctx, srv.URL+"/ok", &v); err == nil {
		t.Error("empty user agent: want error")
	}
}

func TestCache(t *testing.T) {
	var full, notModified atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fresh":
			w.Header().Set("Cache-Control", "public, max-age=60")
		case "/etag":
			w.Header().Set("ETag", `W/"v1"`)
			if r.Header.Get("If-None-Match") == `W/"v1"` {
				notModified.Add(1)
				w.WriteHeader(http.StatusNotModified)
				return
			}
		case "/nocache":
			w.Header().Set("Cache-Control", "no-cache")
		}
		full.Add(1)
		w.Write([]byte(`"body"`))
	}))
	defer srv.Close()
	store, err := cache.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(srv.Client(), "test", 0)
	c.Cache = store
	ctx := context.Background()
	get := func(path string, opts ...Option) {
		t.Helper()
		var s string
		if err := c.GetJSON(ctx, srv.URL+path, &s, opts...); err != nil || s != "body" {
			t.Fatalf("%s: %v %q", path, err, s)
		}
	}

	get("/fresh")
	get("/fresh")
	get("/etag")
	get("/etag")
	get("/nocache")
	get("/nocache")
	get("/immutable", MinTTL(time.Hour))
	get("/immutable", MinTTL(time.Hour))
	// fresh 1, etag 1, nocache 2, immutable 1.
	if full.Load() != 5 || notModified.Load() != 1 {
		t.Errorf("full responses = %d, 304s = %d", full.Load(), notModified.Load())
	}
}
