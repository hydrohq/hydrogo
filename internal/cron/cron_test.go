package cron

import (
	"testing"
	"time"
)

func TestSpecMatches(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse("2006-01-02 15:04", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct {
		spec string
		when string
		want bool
	}{
		{"0 2 * * *", "2026-10-21 02:00", true},
		{"0 2 * * *", "2026-10-21 02:01", false},
		{"*/15 * * * *", "2026-10-21 10:45", true},
		{"*/15 * * * *", "2026-10-21 10:50", false},
		{"0 9 * * 1-5", "2026-10-21 09:00", true},  // Wednesday
		{"0 9 * * 1-5", "2026-10-25 09:00", false}, // Sunday
		{"0 9 * * 7", "2026-10-25 09:00", true},    // 7 = Sunday
		{"30 8 1,15 * *", "2026-10-15 08:30", true},
		{"30 8 1,15 * *", "2026-10-16 08:30", false},
		{"0 0 13 * 5", "2026-10-13 00:00", true}, // day OR weekday when both set
		{"0 0 13 * 5", "2026-10-16 00:00", true},
		{"0 0 13 * 5", "2026-10-14 00:00", false},
		{"0-30/10 * * 10 *", "2026-10-21 05:20", true},
		{"0-30/10 * * 10 *", "2026-11-21 05:20", false},
	}
	for _, c := range cases {
		spec, err := Parse(c.spec)
		if err != nil {
			t.Fatalf("%q: %v", c.spec, err)
		}
		if got := spec.Matches(at(c.when)); got != c.want {
			t.Errorf("%q at %s = %v, want %v", c.spec, c.when, got, c.want)
		}
	}
}

func TestParseRejectsBadExpressions(t *testing.T) {
	for _, bad := range []string{"", "* * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "* * * 13 *", "* * * * 8", "a * * * *", "*/0 * * * *", "5-1 * * * *", "* * * * * *"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
