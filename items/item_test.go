package items_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justintout/osrs/items"
)

func testClientMapping(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"examine": "Ale of the gods.","id": 20056,"members": true,"lowalch": 340,"limit": 4,"value": 850,"highalch": 510,"icon": "Ale of the gods.png","name": "Ale of the gods"}
		]`))
	}))
	defer s.Close()

	c := items.NewClient("test", items.ForCustomEndpoint(s.URL))
	items, err := c.Mapping()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Ale of the gods" || items[0].ID != 20056 {
		t.Errorf("did not find item, got: %v", items)
	}
}
