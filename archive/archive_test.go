package archive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justintout/go-osrs/prices"
)

func TestEnsureFetchesOnlyMissingWindows(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		ts, _ := strconv.Atoi(r.URL.Query().Get("timestamp"))
		fmt.Fprintf(w, `{"data":{"4151":{"avgHighPrice":%d,"highPriceVolume":1,"avgLowPrice":%d,"lowPriceVolume":2}}}`, ts%1000, ts%1000)
	}))
	defer srv.Close()
	a, err := Open(filepath.Join(t.TempDir(), "prices.db"), prices.NewClient("test", prices.WithBaseURL(srv.URL)))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ctx := context.Background()
	end := time.Now().Add(-3 * time.Hour).Truncate(time.Hour)

	if n, err := a.Ensure(ctx, prices.OneHour, end.Add(-4*time.Hour), end); err != nil || n != 4 {
		t.Fatalf("first Ensure = %d, %v", n, err)
	}
	// Overlapping range: only the two new hours are fetched.
	if n, err := a.Ensure(ctx, prices.OneHour, end.Add(-6*time.Hour), end); err != nil || n != 2 {
		t.Fatalf("second Ensure = %d, %v", n, err)
	}
	if requests.Load() != 6 {
		t.Errorf("requests = %d, want 6", requests.Load())
	}
	series, err := a.Series(prices.OneHour, end.Add(-6*time.Hour), end)
	if err != nil {
		t.Fatal(err)
	}
	if pts := series[4151]; len(pts) != 6 || !pts[0].Timestamp.Equal(end.Add(-6*time.Hour)) || pts[5].LowPriceVolume != 2 {
		t.Errorf("series = %+v", pts)
	}
}
