package transport

import "testing"

func TestShouldAutoCheckpoint(t *testing.T) {
	tests := []struct {
		count int
		want  bool
	}{
		{count: 1, want: true},
		{count: 2, want: false},
		{count: 9, want: false},
		{count: 10, want: true},
		{count: 11, want: false},
		{count: 20, want: true},
	}

	for _, tc := range tests {
		if got := shouldAutoCheckpoint(tc.count); got != tc.want {
			t.Fatalf("shouldAutoCheckpoint(%d) = %v, want %v", tc.count, got, tc.want)
		}
	}
}
