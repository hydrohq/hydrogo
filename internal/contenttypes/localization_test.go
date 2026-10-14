package contenttypes

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

const localizedGuide = `id: guide
label: Guide
title_field: title
public: true
drafts: true
revisions: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, localized: true}
  - {id: body, label: Body, kind: richtext, localized: true}
  - {id: rank, label: Rank, kind: number, list: true}
`

func TestLocalizedFieldsAreValidated(t *testing.T) {
	for name, content := range map[string]string{
		"boolean":     "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: boolean, localized: true}]\n",
		"relation":    "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: relation, to: x, localized: true}]\n",
		"searchable":  "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: text, searchable: true, localized: true}]\n",
		"unique":      "id: x\nlabel: X\nrevisions: true\nfields: [{id: a, label: A, kind: text, unique: true, localized: true}]\n",
		"no history":  "id: x\nlabel: X\nfields: [{id: a, label: A, kind: text, localized: true}]\n",
		"locale name": "id: x\nlabel: X\nfields: [{id: locale, label: A, kind: text}]\n",
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); err == nil {
			t.Errorf("%s: invalid localized definition accepted", name)
		}
	}
}

func localizedAdmin(t *testing.T, api AdminAPI) http.Handler {
	t.Helper()
	return generatedAdmin(t, api, func(options *adminstudio.GeneratedAdminOptions) {
		options.CanTransition, options.Transition = api.CanTransition, api.Transition
		options.ListRevisions, options.GetRevision = api.ListRevisions, api.GetRevision
		options.CanRestore, options.RestoreRevision = api.CanRestore, api.RestoreRevision
		options.Locales, options.GetLocalized, options.SaveLocalized = api.LocaleOptions, api.GetLocalized, api.SaveLocalized
		options.ListLocalized, options.CanCopyLocale, options.CopyLocale = api.ListLocalized, api.CanCopyLocale, api.CopyLocale
	})
}

func TestLocalizedTypesAreTranslatedInTheAdminAndServedPerLocale(t *testing.T) {
	service, types := newServiceFor(t, localizedGuide)
	locales := NewLocales(true, "EN", []string{"en", "de", "fr"})
	api := AdminAPI{Service: service, Types: types, Locales: locales}
	admin := localizedAdmin(t, api)
	public := publicMux(PublicAPI{Service: service, Types: types, Locales: locales})
	ctx := context.Background()

	if created := adminPost(t, admin, "/admin/resources/guide/new", url.Values{"id": {"x"}, "title": {"Hello"}, "body": {"<p>Hi</p>"}, "rank": {"1"}}); created.Code != http.StatusSeeOther {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	records, err := service.ListRecords(ctx, "guide", 1, 0)
	if err != nil || len(records) != 1 {
		t.Fatal(records, err)
	}
	id := records[0].ID
	base := "/admin/resources/guide/" + id

	status, body := adminGet(t, admin, base+"?locale=de")
	if status != http.StatusOK || !strings.Contains(body, base+"/locale?locale=de") || !strings.Contains(body, `name="locale_version" value="0"`) {
		t.Fatalf("German edit form: %d\n%s", status, body)
	}
	save := func(version string, values url.Values) int {
		values.Set("locale_version", version)
		return adminPost(t, admin, base+"/locale?locale=de", values).Code
	}
	if code := save("0", url.Values{"title": {"Hallo"}, "body": {""}}); code != http.StatusSeeOther {
		t.Fatalf("first German save: %d", code)
	}
	if code := save("0", url.Values{"title": {"Servus"}}); code != http.StatusConflict {
		t.Fatalf("stale German save: %d, want 409", code)
	}
	if code := save("1", url.Values{"title": {"Hallo Welt"}, "body": {""}}); code != http.StatusSeeOther {
		t.Fatalf("second German save: %d", code)
	}
	// French is versioned on its own.
	if code := adminPost(t, admin, base+"/locale?locale=fr", url.Values{"locale_version": {"0"}, "title": {"Bonjour"}}).Code; code != http.StatusSeeOther {
		t.Fatalf("French save: %d", code)
	}
	if record, _ := service.GetRecord(ctx, "guide", id); record.Version != 1 || record.Data["title"] != "Hello" {
		t.Fatalf("translations changed the record: %+v", record)
	}
	if _, body := adminGet(t, admin, "/admin/resources/guide?locale=de"); !strings.Contains(body, "Hallo Welt") {
		t.Fatalf("German list lacks the translation:\n%s", body)
	}

	// Drafts never leak, whatever the locale.
	if count := publicCount(t, public, "/api/v1/content/guide?locale=de"); count != 0 {
		t.Fatalf("draft listed in German: %d", count)
	}
	if recorder, _ := call(t, public, http.MethodGet, "/api/v1/content/guide/"+id+"?locale=de", ""); recorder.Code != http.StatusNotFound {
		t.Fatalf("draft served in German: %d", recorder.Code)
	}
	if published := adminPost(t, admin, base+"/workflow", url.Values{"target": {"published"}, "expected_version": {"1"}}); published.Code != http.StatusSeeOther {
		t.Fatalf("publish: %d %s", published.Code, published.Body)
	}

	_, german := call(t, public, http.MethodGet, "/api/v1/content/guide/"+id+"?locale=DE", "")
	if german["title"] != "Hallo Welt" || german["body"] != "<p>Hi</p>" || german["locale"] != "de" || german["status"] != StatusPublished {
		t.Fatalf("German record (body falls back to English): %v", german)
	}
	_, english := call(t, public, http.MethodGet, "/api/v1/content/guide/"+id, "")
	if english["title"] != "Hello" || english["locale"] != "en" {
		t.Fatalf("default-locale record: %v", english)
	}
	_, list := call(t, public, http.MethodGet, "/api/v1/content/guide?locale=fr", "")
	items := list["items"].([]any)
	if list["locale"] != "fr" || len(items) != 1 || items[0].(map[string]any)["title"] != "Bonjour" || items[0].(map[string]any)["locale"] != "fr" {
		t.Fatalf("French list: %v", list)
	}
	for _, target := range []string{"/api/v1/content/guide?locale=es", "/api/v1/content/guide/" + id + "?locale=es", "/api/v1/content/guide?locale=de&locale=fr"} {
		if recorder, _ := call(t, public, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d, want 400", target, recorder.Code)
		}
	}
	if moved := adminPost(t, admin, base+"/workflow", url.Values{"target": {"draft"}, "expected_version": {"2"}}); moved.Code != http.StatusSeeOther {
		t.Fatalf("unpublish: %d", moved.Code)
	}
	if recorder, _ := call(t, public, http.MethodGet, "/api/v1/content/guide/"+id+"?locale=de", ""); recorder.Code != http.StatusNotFound {
		t.Fatalf("unpublished record served in German: %d", recorder.Code)
	}

	// Copy English into German: only offered into non-default locales.
	if copied := adminPost(t, admin, base+"/locale-copy?locale=de", url.Values{"from_locale": {"en"}, "expected_version": {"2"}}); copied.Code != http.StatusSeeOther {
		t.Fatalf("copy: %d %s", copied.Code, copied.Body)
	}
	if values, version, err := api.GetLocalized(httptestRequest(), adminCollection(t, types), id, "de"); err != nil || version != 3 || values["title"] != "Hello" || values["body"] != "<p>Hi</p>" {
		t.Fatalf("after copy: %v %d %v", values, version, err)
	}
	intoDefault, _ := http.NewRequest(http.MethodPost, base+"/locale-copy?locale=en", nil)
	if api.CanCopyLocale(intoDefault, adminCollection(t, types), id, "de") {
		t.Fatal("copy into the default locale offered")
	}

	// Revisions cover the default-locale record only.
	if revisions, err := service.ListRevisions(ctx, "guide", id, 50); err != nil || len(revisions) != 3 {
		t.Fatalf("revisions: %d %v", len(revisions), err)
	}
	if err := service.DeleteRecord(ctx, "guide", id); err != nil {
		t.Fatal(err)
	}
}

func TestLocalizedFieldsArePlainWithOneLocale(t *testing.T) {
	service, types := newServiceFor(t, localizedGuide)
	locales := NewLocales(false, "en", []string{"en", "de"})
	if locales.Enabled() {
		t.Fatal("disabled i18n enabled localization")
	}
	// Without locale callbacks the admin must still register: the flag is a
	// no-op for a single-locale site.
	api := AdminAPI{Service: service, Types: types, Locales: locales}
	admin := generatedAdmin(t, api, func(options *adminstudio.GeneratedAdminOptions) {
		options.CanTransition, options.Transition = api.CanTransition, api.Transition
		options.ListRevisions, options.GetRevision = api.ListRevisions, api.GetRevision
		options.CanRestore, options.RestoreRevision = api.CanRestore, api.RestoreRevision
	})
	if status, body := adminGet(t, admin, "/admin/resources/guide/new"); status != http.StatusOK || strings.Contains(body, "locale_version") {
		t.Fatalf("single-locale form: %d", status)
	}
	public := publicMux(PublicAPI{Service: service, Types: types, Locales: locales})
	if recorder, body := call(t, public, http.MethodGet, "/api/v1/content/guide?locale=en", ""); recorder.Code != http.StatusOK || body["locale"] != "en" {
		t.Fatalf("default locale: %d %v", recorder.Code, body)
	}
	if recorder, _ := call(t, public, http.MethodGet, "/api/v1/content/guide?locale=de", ""); recorder.Code != http.StatusBadRequest {
		t.Fatalf("disabled locale: %d, want 400", recorder.Code)
	}
}
