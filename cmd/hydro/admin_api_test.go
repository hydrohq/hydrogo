package main

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
)

type fakeUserAdmin struct{ users []*auth.User }

func (f fakeUserAdmin) List(context.Context) ([]*auth.User, error) { return f.users, nil }
func (fakeUserAdmin) UpdateEmail(context.Context, database.Executor, string, string) error {
	return nil
}
func (fakeUserAdmin) UpdateRole(context.Context, string, auth.Role) error { return nil }

func adminAPIRequest(method, target string, body string, user *auth.User) *http.Request {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	request := httptest.NewRequest(method, target, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if user != nil {
		request = request.WithContext(auth.WithUser(request.Context(), user))
	}
	return request
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	return body
}

func TestAdminDashboardShape(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"pages": true, "news": false, "push": true}}
	editor := &auth.User{ID: "u1", Email: "editor@example.test", Role: auth.RoleEditor, Active: true}

	anonymous := httptest.NewRecorder()
	app.adminDashboardHandler(anonymous, adminAPIRequest(http.MethodGet, "/api/v1/admin/dashboard", "", nil))
	if anonymous.Code != http.StatusForbidden {
		t.Fatalf("anonymous dashboard: %d", anonymous.Code)
	}

	recorder := httptest.NewRecorder()
	app.adminDashboardHandler(recorder, adminAPIRequest(http.MethodGet, "/api/v1/admin/dashboard", "", editor))
	if recorder.Code != http.StatusOK {
		t.Fatalf("dashboard: %d %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data adminDashboard `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.User.Email != "editor@example.test" || !body.Data.Modules["pages"] || body.Data.Modules["news"] || !body.Data.PushEnabled || body.Data.Runtime != nil {
		t.Fatalf("dashboard=%+v", body.Data)
	}
	if strings.Contains(recorder.Body.String(), "password_hash") {
		t.Fatal("dashboard leaks the password hash")
	}
}

func TestAdminAuditRejectsInvalidFilter(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"pages": true}}
	admin := &auth.User{ID: "a1", Role: auth.RoleAdmin, Active: true}

	forbidden := httptest.NewRecorder()
	app.adminAuditHandler(forbidden, adminAPIRequest(http.MethodGet, "/api/v1/admin/audit", "", &auth.User{ID: "e1", Role: auth.RoleEditor, Active: true}))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("editor audit: %d", forbidden.Code)
	}

	recorder := httptest.NewRecorder()
	app.adminAuditHandler(recorder, adminAPIRequest(http.MethodGet, "/api/v1/admin/audit?from=yesterday&limit=500", "", admin))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid filter: %d %s", recorder.Code, recorder.Body)
	}
	body := decodeEnvelope(t, recorder)
	fields, _ := body["fields"].(map[string]any)
	if fields["from"] != "invalid" || fields["limit"] != "invalid" || body["request_id"] == nil {
		t.Fatalf("body=%v", body)
	}
}

func TestAdminUsersListRequiresUsersManage(t *testing.T) {
	app := &serveApp{userAdmin: fakeUserAdmin{users: []*auth.User{{ID: "u1", Email: "one@example.test", Role: auth.RoleEditor, Active: true}}}}

	forbidden := httptest.NewRecorder()
	app.adminUsersListHandler(forbidden, adminAPIRequest(http.MethodGet, "/api/v1/admin/users", "", &auth.User{ID: "e1", Role: auth.RoleEditor, Active: true}))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("editor users: %d", forbidden.Code)
	}

	recorder := httptest.NewRecorder()
	app.adminUsersListHandler(recorder, adminAPIRequest(http.MethodGet, "/api/v1/admin/users", "", &auth.User{ID: "a1", Role: auth.RoleAdmin, Active: true}))
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin users: %d %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data []adminUserItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].Email != "one@example.test" || len(body.Data[0].Permissions) == 0 {
		t.Fatalf("users=%+v", body.Data)
	}
	for _, permission := range body.Data[0].Permissions {
		if permission == auth.PermissionUsersManage {
			t.Fatal("editor granted users.manage")
		}
	}

	permissions := httptest.NewRecorder()
	app.adminPermissionsHandler(permissions, adminAPIRequest(http.MethodGet, "/api/v1/admin/permissions", "", &auth.User{ID: "a1", Role: auth.RoleAdmin, Active: true}))
	if permissions.Code != http.StatusOK || !strings.Contains(permissions.Body.String(), `"role_defaults"`) {
		t.Fatalf("permissions: %d %s", permissions.Code, permissions.Body)
	}
}

func TestAdminClientErrorsRateLimited(t *testing.T) {
	app := &serveApp{}
	// Three reports per window; the rejected one below counts as well.
	handler := app.adminClientErrorHandler(app.newRateLimiter(3, time.Minute))
	user := &auth.User{ID: "u1", Role: auth.RoleEditor, Active: true}

	invalid := httptest.NewRecorder()
	handler(invalid, adminAPIRequest(http.MethodPost, "/api/v1/admin/client-errors", `{"message":"  "}`, user))
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty message: %d", invalid.Code)
	}
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler(recorder, adminAPIRequest(http.MethodPost, "/api/v1/admin/client-errors", `{"message":"TypeError: x is undefined","url":"/admin/pages"}`, user))
		if recorder.Code != http.StatusAccepted {
			t.Fatalf("report %d: %d %s", i, recorder.Code, recorder.Body)
		}
	}
	limited := httptest.NewRecorder()
	handler(limited, adminAPIRequest(http.MethodPost, "/api/v1/admin/client-errors", `{"message":"again"}`, user))
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit: %d", limited.Code)
	}
}

func TestAdminSystemInfoAndAccountLocale(t *testing.T) {
	app := &serveApp{devMode: true, modules: map[string]bool{"pages": true}}
	user := &auth.User{ID: "u1", Role: auth.RoleViewer, Active: true}

	info := httptest.NewRecorder()
	app.adminSystemInfoHandler(info, adminAPIRequest(http.MethodGet, "/api/v1/admin/system/info", "", user))
	if info.Code != http.StatusOK {
		t.Fatalf("system info: %d %s", info.Code, info.Body)
	}
	var body struct {
		Data adminSystemInfo `json:"data"`
	}
	if err := json.Unmarshal(info.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Version != version || !body.Data.DevMode || !body.Data.Modules["pages"] {
		t.Fatalf("info=%+v", body.Data)
	}

	metrics := httptest.NewRecorder()
	app.adminSystemMetricsHandler(metrics, adminAPIRequest(http.MethodGet, "/api/v1/admin/system/metrics", "", user))
	if metrics.Code != http.StatusForbidden {
		t.Fatalf("viewer metrics: %d", metrics.Code)
	}

	invalid := httptest.NewRecorder()
	app.adminAccountLocaleHandler(invalid, adminAPIRequest(http.MethodPut, "/api/v1/admin/account/locale", `{"locale":"xx"}`, user))
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid locale: %d", invalid.Code)
	}
	valid := httptest.NewRecorder()
	app.adminAccountLocaleHandler(valid, adminAPIRequest(http.MethodPut, "/api/v1/admin/account/locale", `{"locale":"en"}`, user))
	if valid.Code != http.StatusOK || !strings.Contains(valid.Header().Get("Set-Cookie"), adminLocaleCookieName+"=en") {
		t.Fatalf("locale: %d %s %q", valid.Code, valid.Body, valid.Header().Get("Set-Cookie"))
	}

	cleared := httptest.NewRecorder()
	app.adminAccountLocaleHandler(cleared, adminAPIRequest(http.MethodPut, "/api/v1/admin/account/locale", `{"locale":""}`, user))
	if cleared.Code != http.StatusOK || !strings.Contains(cleared.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("clear locale: %d %s %q", cleared.Code, cleared.Body, cleared.Header().Get("Set-Cookie"))
	}

	catalog := httptest.NewRecorder()
	request := adminAPIRequest(http.MethodGet, "/api/v1/admin/i18n/en", "", user)
	request.SetPathValue("locale", "en")
	app.adminI18nHandler(catalog, request)
	if catalog.Code != http.StatusOK {
		t.Fatalf("i18n: %d %s", catalog.Code, catalog.Body)
	}
	messages, _ := decodeEnvelope(t, catalog)["data"].(map[string]any)
	if len(messages) == 0 {
		t.Fatal("empty catalog")
	}
	unknown := httptest.NewRecorder()
	request = adminAPIRequest(http.MethodGet, "/api/v1/admin/i18n/xx", "", user)
	request.SetPathValue("locale", "xx")
	app.adminI18nHandler(unknown, request)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown locale: %d", unknown.Code)
	}
}
