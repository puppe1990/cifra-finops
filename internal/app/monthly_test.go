package app

import "testing"

func TestMonthlySyncHourFromEnv(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 23},
		{"0", 0},
		{"7", 7},
		{"23", 23},
		{"24", 23},
		{"-1", 23},
		{"noon", 23},
	}
	for _, tc := range cases {
		if got := MonthlySyncHourFromEnv(tc.raw); got != tc.want {
			t.Fatalf("MonthlySyncHourFromEnv(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}
