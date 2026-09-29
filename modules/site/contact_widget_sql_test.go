package site_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	"github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestSaveContactWidgetPersistsPublicConfiguration(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "contact-widget.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	service := site.Service{Transactions: database.NewTransactor(db), Store: store}
	want := site.ContactWidget{Enabled: true, Phone: "+90 212 555 0100", WhatsAppNumber: "+90 555 123 4567", Message: "Merhaba", Position: "bottom-left", MobileOnly: true}
	if err := service.SaveContactWidget(ctx, want); err != nil {
		t.Fatal(err)
	}
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got := site.ContactWidgetFromSettings(settings); got != want {
		t.Fatalf("saved widget = %#v, want %#v", got, want)
	}
	if err := service.SaveContactWidget(ctx, site.ContactWidget{Enabled: true, Position: "bottom-right"}); err == nil {
		t.Fatal("invalid widget was saved")
	}
	settings, err = store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got := site.ContactWidgetFromSettings(settings); got != want {
		t.Fatalf("invalid update changed saved widget: %#v", got)
	}
	wantProfiles := []site.SocialProfile{{Platform: "instagram", URL: "https://instagram.com/hydrogo"}, {Platform: "github", URL: "https://github.com/hydrogo"}}
	if err := service.SaveSocialProfiles(ctx, wantProfiles); err != nil {
		t.Fatal(err)
	}
	settings, err = store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got := site.SocialProfilesFromSettings(settings); !reflect.DeepEqual(got, wantProfiles) {
		t.Fatalf("saved profiles = %#v, want %#v", got, wantProfiles)
	}
	if err := service.SaveSocialProfiles(ctx, []site.SocialProfile{{Platform: "instagram", URL: "https://attacker.test"}}); err == nil {
		t.Fatal("invalid social profile was saved")
	}
	settings, err = store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got := site.SocialProfilesFromSettings(settings); !reflect.DeepEqual(got, wantProfiles) {
		t.Fatalf("invalid update changed profiles: %#v", got)
	}
}
