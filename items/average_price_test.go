package items_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/osrs/items"
)

func testClientFiveMinute(t *testing.T) {
	s := avgTestServer()
	c := items.NewClient("testing", items.ForCustomEndpoint(s.URL))
	a, err := c.FiveMinute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	aa, ok := a[20056]
	if !ok {
		t.Errorf("target item missing in response: %v", a)
	}
	if aa.AvgHighPrice != 279995 || aa.LowPriceVolume != 3 {
		t.Errorf("target item not decoded correctly: %v", aa)
	}
}

// func testClientFiveMinuteStartingAt(t *testing.T) {

// }

// func testClientOneHour(t *testing.T) {

// }

// func testClientOneHourStartingAt(t *testing.T) {}

func avgTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data": {"20056": {
      "avgHighPrice": 279995,
      "highPriceVolume": 3,
      "avgLowPrice": 271491,
      "lowPriceVolume": 3
    }}}`))
	}))
}
