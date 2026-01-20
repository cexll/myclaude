package utils

import "testing"

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a less than b", 1, 2, 1},
		{"b less than a", 5, 3, 3},
		{"equal values", 7, 7, 7},
		{"negative a", -5, 3, -5},
		{"negative b", 5, -3, -3},
		{"both negative", -5, -3, -5},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -5, -5},
		{"large values", 1000000, 999999, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func BenchmarkMin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Min(i, i+1)
	}
}
