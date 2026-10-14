package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func runTranslationContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	schema := collections.Schema{ID: "localized", Label: "Localized", Fields: []collections.Field{
		{ID: "title", Label: "Title", Kind: collections.Text, Required: true, Localized: true},
		{ID: "body", Label: "Body", Kind: collections.RichText, Localized: true},
		{ID: "rank", Label: "Rank", Kind: collections.Number},
	}}
	plain := collections.Schema{ID: "unlocalized", Label: "Unlocalized", Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text}}}
	for _, item := range []collections.Schema{schema, plain} {
		if err := service.CreateSchema(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_ = service.DeleteSchema(ctx, schema.ID)
		_ = service.DeleteSchema(ctx, plain.ID)
	}()
	record, err := service.CreateRecord(ctx, schema.ID, map[string]any{"title": "Hello", "body": "<p>Body</p>", "rank": 1})
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.CreateRecord(ctx, schema.ID, map[string]any{"title": "Other"})
	if err != nil {
		t.Fatal(err)
	}

	empty, err := service.GetTranslation(ctx, schema.ID, record.ID, "de")
	if err != nil || empty.Version != 0 || len(empty.Data) != 0 {
		t.Fatalf("%s: untranslated locale = %+v, %v", dialect, empty, err)
	}
	saved, err := service.SaveTranslation(ctx, schema.ID, record.ID, "de", map[string]any{"title": "Hallo", "body": ""}, 0)
	if err != nil || saved.Version != 1 {
		t.Fatalf("%s: first save = %+v, %v", dialect, saved, err)
	}
	// A second editor that loaded the empty locale must not overwrite it.
	if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "de", map[string]any{"title": "Servus"}, 0); !errors.Is(err, collections.ErrVersionConflict) {
		t.Fatalf("%s: stale insert err = %v", dialect, err)
	}
	if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "de", map[string]any{"title": "Hallo!", "body": "<p>Text</p>"}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "de", map[string]any{"title": "Too late"}, 1); !errors.Is(err, collections.ErrVersionConflict) {
		t.Fatalf("%s: stale update err = %v", dialect, err)
	}
	// Locales are versioned independently.
	if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "fr", map[string]any{"title": "Bonjour"}, 0); err != nil {
		t.Fatal(err)
	}
	german, err := service.GetTranslation(ctx, schema.ID, record.ID, "de")
	if err != nil || german.Version != 2 || german.Data["title"] != "Hallo!" || german.Data["body"] != "<p>Text</p>" || german.UpdatedAt.IsZero() {
		t.Fatalf("%s: german = %+v, %v", dialect, german, err)
	}
	stored, err := service.GetRecord(ctx, schema.ID, record.ID)
	if err != nil || stored.Data["title"] != "Hello" || stored.Version != 1 {
		t.Fatalf("%s: translations changed the record: %+v %v", dialect, stored, err)
	}

	for _, data := range []map[string]any{{"rank": 2}, {"unknown": "x"}, {"title": 5}} {
		if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "de", data, 2); !errors.Is(err, collections.ErrInvalidRecord) {
			t.Fatalf("%s: invalid translation %v err = %v", dialect, data, err)
		}
	}
	if _, err := service.SaveTranslation(ctx, schema.ID, record.ID, "DE; drop", map[string]any{"title": "x"}, 0); !errors.Is(err, collections.ErrInvalidLocale) {
		t.Fatalf("%s: invalid locale err = %v", dialect, err)
	}
	if _, err := service.SaveTranslation(ctx, schema.ID, "r_missing", "de", map[string]any{"title": "x"}, 0); !errors.Is(err, collections.ErrRecordNotFound) {
		t.Fatalf("%s: missing record err = %v", dialect, err)
	}
	plainRecord, err := service.CreateRecord(ctx, plain.ID, map[string]any{"title": "Plain"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetTranslation(ctx, plain.ID, plainRecord.ID, "de"); !errors.Is(err, collections.ErrNotLocalized) {
		t.Fatalf("%s: unlocalized schema err = %v", dialect, err)
	}

	listed, err := service.ListTranslations(ctx, schema.ID, []string{record.ID, other.ID}, "de")
	if err != nil || len(listed) != 1 || listed[record.ID].Data["title"] != "Hallo!" {
		t.Fatalf("%s: list = %+v, %v", dialect, listed, err)
	}

	runTenantTranslationContract(t, db, dialect, schema)

	if err := service.DeleteRecord(ctx, schema.ID, record.ID); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hydro_collection_translations WHERE record_id=`+Store{Dialect: dialect}.p(1), record.ID).Scan(&left); err != nil || left != 0 {
		t.Fatalf("%s: translations survived their record: %d %v", dialect, left, err)
	}
}

// runTenantTranslationContract checks that a tenant can neither read nor
// write another tenant's translations, even with a known record ID.
func runTenantTranslationContract(t *testing.T, db database.Executor, dialect database.Dialect, schema collections.Schema) {
	t.Helper()
	alpha, err := tenant.NewScope("tenant-alpha", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-beta", "beta")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(context.Background(), alpha)
	betaCtx := tenant.WithScope(context.Background(), beta)
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect, RequireTenantScope: true}}
	for _, ctx := range []context.Context{alphaCtx, betaCtx} {
		if err := service.CreateSchema(ctx, schema); err != nil {
			t.Fatal(err)
		}
		defer func(ctx context.Context) { _ = service.DeleteSchema(ctx, schema.ID) }(ctx)
	}
	record, err := service.CreateRecord(alphaCtx, schema.ID, map[string]any{"title": "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveTranslation(alphaCtx, schema.ID, record.ID, "de", map[string]any{"title": "Alpha DE"}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetTranslation(betaCtx, schema.ID, record.ID, "de"); !errors.Is(err, collections.ErrRecordNotFound) {
		t.Fatalf("%s: cross-tenant translation read err = %v", dialect, err)
	}
	if _, err := service.SaveTranslation(betaCtx, schema.ID, record.ID, "de", map[string]any{"title": "Beta"}, 1); !errors.Is(err, collections.ErrRecordNotFound) {
		t.Fatalf("%s: cross-tenant translation write err = %v", dialect, err)
	}
	if listed, err := service.ListTranslations(betaCtx, schema.ID, []string{record.ID}, "de"); err != nil || len(listed) != 0 {
		t.Fatalf("%s: cross-tenant translation list = %+v, %v", dialect, listed, err)
	}
	if err := service.DeleteRecord(alphaCtx, schema.ID, record.ID); err != nil {
		t.Fatal(err)
	}
}
