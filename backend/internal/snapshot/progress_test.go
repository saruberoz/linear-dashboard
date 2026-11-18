package snapshot

import "testing"

func TestFetchPercent(t *testing.T) {
	tests := []struct {
		name              string
		fetched, expected int64
		want              int32
	}{
		{name: "unknown size on the first sync", fetched: 500, expected: 0, want: -1},
		{name: "start", fetched: 0, expected: 1000, want: 0},
		{name: "half fetched", fetched: 500, expected: 1000, want: 47},
		{name: "all fetched stops short of saving", fetched: 1000, expected: 1000, want: 95},
		{name: "workspace grew", fetched: 1300, expected: 1000, want: 95},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fetchPercent(tt.fetched, tt.expected); got != tt.want {
				t.Errorf("fetchPercent(%d, %d) = %d, want %d", tt.fetched, tt.expected, got, tt.want)
			}
		})
	}
}
