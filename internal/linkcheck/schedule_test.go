package linkcheck

import (
	"testing"
	"time"
)

func TestNextMonthlyScan(t *testing.T) {
	cases := []struct{ now, want string }{
		{"2026-09-29T09:00:00Z", "2026-10-01T03:00:00Z"},
		{"2026-10-01T02:59:59Z", "2026-10-01T03:00:00Z"},
		{"2026-10-01T03:00:00Z", "2026-11-01T03:00:00Z"},
		{"2026-12-31T23:59:59-05:00", "2027-02-01T03:00:00Z"},
	}
	for _, tc := range cases {
		now, err := time.Parse(time.RFC3339, tc.now)
		if err != nil {
			t.Fatal(err)
		}
		if got := NextMonthlyScan(now).Format(time.RFC3339); got != tc.want {
			t.Errorf("NextMonthlyScan(%s)=%s want %s", tc.now, got, tc.want)
		}
	}
}
