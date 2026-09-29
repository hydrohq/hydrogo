package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
)

func TestEmailBrandingStorePersistsAndValidatesSettings(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "mail-branding.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE mail_branding_settings(singleton_id INTEGER PRIMARY KEY,legal_name TEXT NOT NULL,physical_address TEXT NOT NULL,privacy_policy_url TEXT NOT NULL,footer_notice TEXT NOT NULL,logo_asset_id TEXT NOT NULL,primary_color TEXT NOT NULL,email_footer_text TEXT NOT NULL,updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	store := EmailBrandingStore{DB: db, Dialect: database.SQLite}
	want := mailer.EmailBranding{LegalName: "Example Ltd", PhysicalAddress: "1 Main Street", PrivacyPolicyURL: "https://example.test/privacy", FooterNotice: "Terms apply", LogoAssetID: "logo", PrimaryColor: "#abcdef", FooterText: "Unsubscribe details"}
	if err := store.WriteEmailBranding(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadEmailBranding(ctx)
	if err != nil || got != want {
		t.Fatalf("branding=%+v err=%v", got, err)
	}
	if err := store.WriteEmailBranding(ctx, mailer.EmailBranding{PrivacyPolicyURL: "javascript:alert(1)"}); err == nil {
		t.Fatal("unsafe privacy URL accepted")
	}
}
