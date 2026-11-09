package media

import "testing"

// sqlstore refuses list windows above 200; the admin list must stay within it.
func TestAdminListScanLimitStaysWithinTheStoreCap(t *testing.T) {
	if adminListScanLimit > 200 {
		t.Fatalf("adminListScanLimit = %d exceeds the store's cap of 200", adminListScanLimit)
	}
}
