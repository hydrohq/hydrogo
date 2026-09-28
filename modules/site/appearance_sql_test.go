package site_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	"github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestSaveAppearancePersistsPublicTokensTogether(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "appearance.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.Store{Dialect: database.SQLite}
	service := site.Service{Transactions: database.NewTransactor(db), Store: store}
	want := site.Appearance{PrimaryColor: "#155EEF", AccentColor: "#155EEF", DarkPrimaryColor: "#155EEF", DarkAccentColor: "#155EEF", FontFamily: "inter", LogoURL: "/media/logo.svg", DarkLogoURL: "/media/logo-dark.svg", FaviconURL: "/media/favicon.svg", SocialImageURL: "/media/social.webp"}
	if err := service.SaveAppearance(ctx, want); err != nil {
		t.Fatal(err)
	}
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	got := site.AppearanceFromSettings(settings)
	if got != want {
		t.Fatalf("saved appearance = %#v, want %#v", got, want)
	}
	if err := service.SaveAppearance(ctx, site.Appearance{PrimaryColor: "#FFFFFF", AccentColor: "#FFFFFF", DarkPrimaryColor: "#FFFFFF", DarkAccentColor: "#FFFFFF", FontFamily: "inter"}); err == nil {
		t.Fatal("low-contrast appearance was saved")
	}
	settings, err = store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got = site.AppearanceFromSettings(settings); got != want {
		t.Fatalf("invalid appearance changed stored value: %#v", got)
	}
}
