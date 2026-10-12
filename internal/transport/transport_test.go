package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/go-osrs"
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
	c := Client{HTTP: srv.Client(), UserAgent: "test-agent"}
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
