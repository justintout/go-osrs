package ge_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/go-osrs/ge"
)

func TestClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalogue/detail.json":
			w.Write([]byte(`{"item":{"id":4151,"name":"Abyssal whip","members":"true","current":{"trend":"neutral","price":"800.4k"},"today":{"trend":"negative","price":"- 13.5k"},"day30":{"change":"-1.0%"},"day90":{"change":"-21.0%"},"day180":{"change":"+3.5%"}}}`))
		case "/graph/4151.json":
			w.Write([]byte(`{"daily":{"1776384000000":1200000,"1776297600000":1211325},"average":{"1776384000000":1214000,"1776297600000":1215293}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := ge.NewClient("test", ge.WithBaseURL(srv.URL))
	ctx := context.Background()

	item, err := c.Item(ctx, 4151)
	if err != nil {
		t.Fatal(err)
	}
	if !item.Members || item.Price != 800_400 || item.TodayChange != -13_500 || item.Change90d != -21 || item.Change180d != 3.5 {
		t.Errorf("item = %+v", item)
	}

	hist, err := c.History(ctx, 4151)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].Price != 1_211_325 || hist[1].Average != 1_214_000 {
		t.Errorf("history = %+v", hist)
	}
}
