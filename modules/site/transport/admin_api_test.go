package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// adminMux mounts every site route behind a fixed signed-in user.
func adminMux(t *testing.T, handler Handler, user *auth.User) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user != nil {
			r = r.WithContext(auth.WithUser(r.Context(), user))
		}
		mux.ServeHTTP(w, r)
	})
}

func call(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func decodeData(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, response.Body.String())
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode data: %v body=%s", err, response.Body.String())
	}
}

var siteAdmin = &auth.User{ID: "admin", Email: "admin@example.test", Role: auth.RoleAdmin, Active: true}

func TestAdminSettingsRejectsUnknownAndInvalidKeys(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	response := call(t, mux, http.MethodPut, "/api/v1/admin/site/settings", `{"settings":[{"key":"site.custom_code","value":"x"},{"key":"appearance.favicon_url","value":"javascript:alert(1)"},{"key":"site.name","value":"HydroGo"}]}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Code      string            `json:"code"`
		Fields    map[string]string `json:"fields"`
		RequestID string            `json:"request_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Fields["site.custom_code"] != "unknown" || body.Fields["appearance.favicon_url"] != "invalid" || body.Fields["site.name"] != "" {
		t.Fatalf("body=%s err=%v", response.Body.String(), err)
	}
	response = call(t, mux, http.MethodPut, "/api/v1/admin/site/settings", `{"settings":[{"key":"site.name","value":"HydroGo"},{"key":"seo.llms_excluded_paths","value":"/private\n/drafts"}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var settings []settingView
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/settings", ""), &settings)
	values := map[string]string{}
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	if values["site.name"] != "HydroGo" || values["seo.llms_excluded_paths"] != "/private\n/drafts" || values["site.title"] != "" {
		t.Fatalf("settings=%+v", values)
	}
	var public map[string]string
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/site/settings", ""), &public)
	if public["site.name"] != "HydroGo" {
		t.Fatalf("public settings=%+v", public)
	}
}

func TestAdminMaintenanceKeepsPreviewKeyUnlessCleared(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	response := call(t, mux, http.MethodPut, "/api/v1/admin/site/maintenance", `{"enabled":true,"allowlist":["127.0.0.1"],"headline":"Back soon","preview_key":"a-very-long-preview-key"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var view maintenanceView
	decodeData(t, response, &view)
	if !view.Enabled || !view.HasPreviewKey || view.Headline != "Back soon" {
		t.Fatalf("view=%+v", view)
	}
	decodeData(t, call(t, mux, http.MethodPut, "/api/v1/admin/site/maintenance", `{"enabled":false,"allowlist":[]}`), &view)
	if view.Enabled || !view.HasPreviewKey {
		t.Fatalf("preview key dropped on plain update: %+v", view)
	}
	decodeData(t, call(t, mux, http.MethodPut, "/api/v1/admin/site/maintenance", `{"enabled":false,"clear_preview_key":true}`), &view)
	if view.HasPreviewKey {
		t.Fatalf("preview key not cleared: %+v", view)
	}
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/maintenance", `{"enabled":true,"allowlist":["not-an-ip"]}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid allowlist status=%d", response.Code)
	}
}

func TestAdminSocialLinksAndStaffNoticeRoundTrip(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	response := call(t, mux, http.MethodPut, "/api/v1/admin/site/social-links", `{"profiles":[{"platform":"github","url":"https://github.com/hydrogo"},{"platform":"instagram","url":""}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var links socialLinksView
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/social-links", ""), &links)
	if len(links.Profiles) != 1 || links.Profiles[0].Platform != "github" || len(links.Platforms) == 0 {
		t.Fatalf("links=%+v", links)
	}
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/social-links", `{"profiles":[{"platform":"github","url":"https://evil.example/x"}]}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("foreign host accepted: %d", response.Code)
	}
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	response = call(t, mux, http.MethodPut, "/api/v1/admin/site/staff-notice", `{"active":true,"message":"Deploy at 18:00","severity":"warning","expires_at":"`+expires+`"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var notice site.StaffBroadcastNotice
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/staff-notice", ""), &notice)
	if !notice.Active || notice.Severity != "warning" {
		t.Fatalf("notice=%+v", notice)
	}
}

func TestAdminTranslationsOverrideAndReset(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	rows := i18n.CatalogRows("", "en")
	if len(rows) == 0 {
		t.Skip("no catalog keys")
	}
	key := rows[0].Key
	response := call(t, mux, http.MethodPut, "/api/v1/admin/site/translations/en", `{"values":{"`+key+`":"Overridden","nope.key":"x"}}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown key accepted: %d %s", response.Code, response.Body.String())
	}
	response = call(t, mux, http.MethodPut, "/api/v1/admin/site/translations/en", `{"values":{"`+key+`":"Overridden"}}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := i18n.TForTenant("", "en", key); got != "Overridden" {
		t.Fatalf("override not applied: %q", got)
	}
	var view translationsView
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/translations/en", ""), &view)
	found := false
	for _, row := range view.Rows {
		if row.Key == key && row.Translation == "Overridden" {
			found = true
		}
	}
	if !found || view.Locale != "en" {
		t.Fatalf("view=%+v", view)
	}
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/translations/en", `{"values":{"`+key+`":""}}`); response.Code != http.StatusOK {
		t.Fatalf("reset status=%d", response.Code)
	}
	if got := i18n.TForTenant("", "en", key); got == "Overridden" {
		t.Fatalf("override not removed: %q", got)
	}
	if response := call(t, mux, http.MethodGet, "/api/v1/admin/site/translations/xx", ""); response.Code != http.StatusNotFound {
		t.Fatalf("unsupported locale status=%d", response.Code)
	}
}

func TestAdminRedirectsCRUDImportAndNotFound(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	response := call(t, mux, http.MethodPost, "/api/v1/admin/site/redirects", `{"pattern":"/old","target_path":"/new","status":301}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var rule site.RedirectRule
	decodeData(t, response, &rule)
	if rule.ID == "" || rule.Kind != "exact" || rule.Pattern != "/old" {
		t.Fatalf("rule=%+v", rule)
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/redirects", `{"pattern":"old","target_path":"old","status":200}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid rule status=%d", response.Code)
	}
	response = call(t, mux, http.MethodPut, "/api/v1/admin/site/redirects/"+rule.ID, `{"pattern":"/old","kind":"exact","target_path":"/newer","status":302}`)
	if response.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", response.Code, response.Body.String())
	}
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/redirects/missing", `{"pattern":"/a","target_path":"/b"}`); response.Code != http.StatusNotFound {
		t.Fatalf("unknown rule status=%d", response.Code)
	}
	response = call(t, mux, http.MethodPost, "/api/v1/admin/site/redirects/import", `{"rules":[{"pattern":"/a","target_path":"/b","status":301},{"pattern":"/c/*","kind":"wildcard","target_path":"/d/*","status":301}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", response.Code, response.Body.String())
	}
	var rules []site.RedirectRule
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/redirects", ""), &rules)
	if len(rules) != 3 {
		t.Fatalf("rules=%+v", rules)
	}
	var public []publicRedirect
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/site/redirects", ""), &public)
	if len(public) != 3 || public[0].Pattern == "" {
		t.Fatalf("public rules=%+v", public)
	}
	if response := call(t, mux, http.MethodDelete, "/api/v1/admin/site/redirects/"+rule.ID, ""); response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	if err := handler.Service.RecordNotFound(context.Background(), "/missing-page", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	var missing []site.NotFoundPath
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/not-found", ""), &missing)
	if len(missing) != 1 || missing[0].Path != "/missing-page" {
		t.Fatalf("not found=%+v", missing)
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/not-found/convert", `{"path":"/missing-page","target":"/found"}`); response.Code != http.StatusCreated {
		t.Fatalf("convert status=%d body=%s", response.Code, response.Body.String())
	}
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/not-found", ""), &missing)
	if len(missing) != 0 {
		t.Fatalf("converted path still reported: %+v", missing)
	}
}

func formsHandler(t *testing.T) (Handler, http.Handler) {
	t.Helper()
	db, handler, _ := openSiteDB(t)
	handler.Service.Forms = formstore.Store{Executor: db, Dialect: database.SQLite}
	handler.FormsEnabled = true
	handler.AllowedForms = map[string]struct{}{"contact": {}}
	return handler, adminMux(t, handler, siteAdmin)
}

func TestAdminFormsCRUD(t *testing.T) {
	_, mux := formsHandler(t)
	form := `{"id":"contact","title":"Contact","fields":[{"id":"email","type":"email","label":"E-mail","required":true},{"id":"message","type":"textarea","label":"Message","required":true}]}`
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/forms", form); response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/forms", form); response.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d", response.Code)
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/forms", `{"id":"Bad ID","title":"x","fields":[]}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid form status=%d", response.Code)
	}
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/forms/contact", `{"id":"contact","title":"Contact us","fields":[{"id":"email","type":"email","label":"E-mail","required":true}]}`); response.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", response.Code, response.Body.String())
	}
	var listed []map[string]any
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/forms", ""), &listed)
	if len(listed) != 1 || listed[0]["title"] != "Contact us" {
		t.Fatalf("forms=%+v", listed)
	}
	var public publicFormView
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/forms/contact", ""), &public)
	if public.ID != "contact" || len(public.Fields) != 1 {
		t.Fatalf("public form=%+v", public)
	}
	if response := call(t, mux, http.MethodDelete, "/api/v1/admin/site/forms/contact", ""); response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	if response := call(t, mux, http.MethodGet, "/api/v1/admin/site/forms/contact", ""); response.Code != http.StatusNotFound {
		t.Fatalf("deleted form status=%d", response.Code)
	}
}

func TestPublicSubmitAndAdminInbox(t *testing.T) {
	_, mux := formsHandler(t)
	submit := `{"values":{"email":"user@example.com","name":"Ada","message":"Hello"},"idempotency_key":"submission-token-0001"}`
	response := call(t, mux, http.MethodPost, "/api/v1/forms/contact/submit", submit)
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d body=%s", response.Code, response.Body.String())
	}
	var accepted struct {
		ID       string `json:"id"`
		Replayed bool   `json:"replayed"`
	}
	decodeData(t, response, &accepted)
	if accepted.ID == "" || accepted.Replayed {
		t.Fatalf("accepted=%+v", accepted)
	}
	decodeData(t, call(t, mux, http.MethodPost, "/api/v1/forms/contact/submit", submit), &accepted)
	if !accepted.Replayed {
		t.Fatalf("idempotent replay not detected: %+v", accepted)
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/forms/contact/submit", `{"values":{"name":"Ada"},"idempotency_key":"submission-token-0002"}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing email status=%d body=%s", response.Code, response.Body.String())
	}
	response = call(t, mux, http.MethodGet, "/api/v1/admin/site/inbox?form=contact&limit=10", "")
	if response.Code != http.StatusOK {
		t.Fatalf("inbox status=%d body=%s", response.Code, response.Body.String())
	}
	var items []inboxItem
	decodeData(t, response, &items)
	if len(items) != 1 || items[0].Values["message"] != "Hello" || items[0].Status != "new" {
		t.Fatalf("inbox=%+v", items)
	}
	id := items[0].ID
	response = call(t, mux, http.MethodPatch, "/api/v1/admin/site/inbox/"+id, `{"status":"triaged","stage":"contacted","assigned_to":"admin"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", response.Code, response.Body.String())
	}
	var item inboxItem
	decodeData(t, response, &item)
	if item.Status != "triaged" || item.LeadStage != "contacted" || item.AssignedTo != "admin" {
		t.Fatalf("item=%+v", item)
	}
	if response := call(t, mux, http.MethodPatch, "/api/v1/admin/site/inbox/"+id, `{"status":"bogus"}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status accepted: %d", response.Code)
	}
	if response := call(t, mux, http.MethodPost, "/api/v1/admin/site/inbox/"+id+"/notes", `{"text":"Called back"}`); response.Code != http.StatusCreated {
		t.Fatalf("note status=%d body=%s", response.Code, response.Body.String())
	}
	var notes []site.LeadNote
	decodeData(t, call(t, mux, http.MethodGet, "/api/v1/admin/site/inbox/"+id+"/notes", ""), &notes)
	if len(notes) != 1 || notes[0].Note != "Called back" || notes[0].AuthorID != "admin" {
		t.Fatalf("notes=%+v", notes)
	}
	if response := call(t, mux, http.MethodGet, "/api/v1/admin/site/inbox/missing", ""); response.Code != http.StatusNotFound {
		t.Fatalf("missing item status=%d", response.Code)
	}
	if response := call(t, mux, http.MethodGet, "/api/v1/admin/site/inbox/export.csv?form=contact", ""); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "user@example.com") {
		t.Fatalf("csv status=%d body=%s", response.Code, response.Body.String())
	}
	before := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	var purged map[string]int64
	decodeData(t, call(t, mux, http.MethodPost, "/api/v1/admin/site/inbox/purge", `{"before":"`+before+`"}`), &purged)
	if purged["purged"] != 1 {
		t.Fatalf("purged=%+v", purged)
	}
}

func TestAdminSiteEndpointsRequirePermission(t *testing.T) {
	handler, _ := formsHandler(t)
	viewer := adminMux(t, handler, &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true})
	for _, path := range []string{"/api/v1/admin/site/settings", "/api/v1/admin/site/navigation/tr", "/api/v1/admin/site/redirects", "/api/v1/admin/site/forms", "/api/v1/admin/site/inbox", "/api/v1/admin/site/translations/tr"} {
		if response := call(t, viewer, http.MethodGet, path, ""); response.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d", path, response.Code)
		}
	}
}

func TestAdminDefaultLocaleSettingAcceptsOnlySupportedLanguages(t *testing.T) {
	_, handler, _ := openSiteDB(t)
	mux := adminMux(t, handler, siteAdmin)
	if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/settings", `{"settings":[{"key":"admin.default_locale","value":"xx"}]}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unsupported language status=%d body=%s", response.Code, response.Body.String())
	}
	for _, value := range []string{"tr", ""} {
		if response := call(t, mux, http.MethodPut, "/api/v1/admin/site/settings", `{"settings":[{"key":"admin.default_locale","value":"`+value+`"}]}`); response.Code != http.StatusOK {
			t.Fatalf("value %q status=%d body=%s", value, response.Code, response.Body.String())
		}
	}
}
