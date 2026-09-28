package preview

import (
	"testing"
	"time"
)

func TestPreviewTokenBindsPageTenantRevisionAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	signer := Signer{Key: []byte("0123456789abcdef0123456789abcdef")}
	token, err := signer.Issue(Claims{PageID: "draft-1", TenantID: "tenant-a", Revision: 4}, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.Verify(token, now.Add(time.Minute))
	if err != nil || claims.PageID != "draft-1" || claims.TenantID != "tenant-a" || claims.Revision != 4 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	if _, err := signer.Verify(token, now.Add(2*time.Hour)); err == nil {
		t.Fatal("expired link accepted")
	}
	if _, err := signer.Verify(token+"x", now); err == nil {
		t.Fatal("tampered link accepted")
	}
}
