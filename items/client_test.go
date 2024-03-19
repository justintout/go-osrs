package items_test

import "testing"

func TestClient(t *testing.T) {
	t.Run("Client#Mapping", testClientMapping)
	t.Run("Client#FiveMinute", testClientFiveMinute)
}
