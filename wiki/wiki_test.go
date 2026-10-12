package wiki_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/go-osrs/wiki"
)

func TestQueryString(t *testing.T) {
	got := wiki.Bucket("infobox_item").Select("page_name", "weight").Where("page_name", `Say "hi"`).Where("value", 5).Limit(10).Offset(20).String()
	want := `bucket("infobox_item").select("page_name","weight").where("page_name","Say \"hi\"").where("value",5).limit(10).offset(20).run()`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestParseRarity(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"1/5,000", 0.0002, true},
		{"3/128", 3.0 / 128, true},
		{"~1/100", 0.01, true},
		{"Always", 1, true},
		{"Rare", 0, false},
		{"1/0", 0, false},
	}
	for _, tt := range tests {
		if got, ok := wiki.ParseRarity(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("ParseRarity(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDrops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := `bucket("dropsline").select("page_name","drop_json","rare_drop_table").where("page_name","Vorkath").limit(5000).run()`
		if r.URL.Query().Get("smaxage") != "3600" {
			t.Errorf("smaxage = %q", r.URL.Query().Get("smaxage"))
		}
		if got := r.URL.Query().Get("query"); got != want {
			t.Errorf("query = %s", got)
		}
		w.Write([]byte(`{"bucket":[
			{"drop_json":"{\"Dropped item\":\"Vorki\",\"Dropped from\":\"Vorkath#Post-quest\",\"Rarity\":\"1/3,000\",\"Quantity Low\":1,\"Quantity High\":1,\"Rolls\":1}"},
			{"drop_json":"{\"Dropped item\":\"Rune spear\",\"Dropped from\":\"Vorkath#Post-quest\",\"Rarity\":\"Rare\",\"Quantity Low\":1,\"Quantity High\":1,\"Rolls\":1}","rare_drop_table":""}
		]}`))
	}))
	defer srv.Close()

	drops, err := wiki.NewClient("test", wiki.WithAPIURL(srv.URL)).Drops(context.Background(), "Vorkath")
	if err != nil {
		t.Fatal(err)
	}
	if len(drops) != 2 || drops[0].Item != "Vorki" || drops[0].Probability != 1.0/3000 || drops[0].RareDropTable {
		t.Errorf("drops[0] = %+v", drops[0])
	}
	if !drops[1].RareDropTable || drops[1].Probability != 0 {
		t.Errorf("drops[1] = %+v", drops[1])
	}
}
