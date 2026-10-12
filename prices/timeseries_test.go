package prices_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/go-osrs/prices"
)

func TestTimeseries(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/timeseries" || r.URL.Query().Get("timestep") != "1h" || r.URL.Query().Get("id") != "4151" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Write([]byte(`{"itemId": 4151, "data": [{"timestamp": 1700000000, "avgHighPrice": 1500000, "avgLowPrice": null, "highPriceVolume": 12, "lowPriceVolume": 0}]}`))
	}))
	defer s.Close()

	ts, err := prices.NewClient("test", prices.WithBaseURL(s.URL)).Timeseries(context.Background(), 4151, prices.OneHour)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts.Data) != 1 || ts.Data[0].AvgHighPrice != 1500000 || ts.Data[0].Timestamp.Unix() != 1700000000 {
		t.Errorf("decoded %+v", ts)
	}
}
