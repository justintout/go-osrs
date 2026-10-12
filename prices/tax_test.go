package prices

import "testing"

func TestTax(t *testing.T) {
	const whip, bond = 4151, 13190
	tests := []struct{ id, price, want int }{
		{whip, 49, 0},
		{whip, 50, 1},
		{whip, 1_500_000, 30_000},
		{whip, 1_000_000_000, TaxCap},
		{bond, 10_000_000, 0},
	}
	for _, tt := range tests {
		if got := Tax(tt.id, tt.price); got != tt.want {
			t.Errorf("Tax(%d, %d) = %d, want %d", tt.id, tt.price, got, tt.want)
		}
	}
}
