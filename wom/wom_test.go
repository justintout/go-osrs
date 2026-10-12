package wom_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/go-osrs/wom"
)

func TestClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "secret" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		switch r.URL.EscapedPath() {
		case "/players/lynx%20titan":
			w.Write([]byte(`{"id":1057,"username":"lynx titan","type":"regular","exp":4600000000,"ehp":12164.17,"latestSnapshot":{"data":{"skills":{"attack":{"metric":"attack","experience":200000000,"rank":15,"level":99,"ehp":0}},"bosses":{"vorkath":{"metric":"vorkath","kills":-1,"rank":-1,"ehb":0}}}}}`))
		case "/players/lynx%20titan/gained":
			if r.URL.Query().Get("period") != "week" {
				t.Errorf("period = %q", r.URL.Query().Get("period"))
			}
			w.Write([]byte(`{"startsAt":"2026-10-05T00:00:00Z","data":{"skills":{"overall":{"metric":"overall","experience":{"gained":5,"start":10,"end":15}}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := wom.NewClient("test", wom.WithBaseURL(srv.URL), wom.WithAPIKey("secret"))
	ctx := context.Background()

	p, err := c.Player(ctx, "lynx titan")
	if err != nil {
		t.Fatal(err)
	}
	if p.Exp != 4_600_000_000 || p.LatestSnapshot.Data.Skills["attack"].Level != 99 || p.LatestSnapshot.Data.Bosses["vorkath"].Kills != -1 {
		t.Errorf("player = %+v", p)
	}

	g, err := c.Gains(ctx, "lynx titan", wom.Week)
	if err != nil {
		t.Fatal(err)
	}
	if g.Data.Skills["overall"].Experience != (wom.Delta[int64]{Gained: 5, Start: 10, End: 15}) {
		t.Errorf("gains = %+v", g)
	}

	if p, err := c.Player(ctx, "nobody"); err == nil || p != nil {
		t.Errorf("missing player = %v, %v", p, err)
	}
}
