package externaledition

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	_ "modernc.org/sqlite"
)

func TestExternalEditionGeneratedDraftPaginationUsesStableSQLitePages(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE edition_drafts (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version INTEGER NOT NULL, published_version INTEGER NOT NULL, state TEXT NOT NULL, snapshot TEXT NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY(resource_id, locale))`); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 27; i++ {
		id := fmt.Sprintf("draft-%02d", i)
		snapshot, err := json.Marshal(map[string]any{"title": fmt.Sprintf("Draft %02d", i)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(context.Background(), `INSERT INTO edition_drafts (resource_id,locale,version,published_version,state,snapshot,updated_at) VALUES (?,?,1,0,'draft',?,'2026-10-10T00:00:00Z')`, id, "fr", string(snapshot)); err != nil {
			t.Fatal(err)
		}
	}

	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{{
		ID: "article", Label: "Articles", TitleField: "title", Versioned: true, Revisions: true, Drafts: true, Localization: true,
		Fields:   []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText, Localized: true, VisibleInList: true}, {ID: "status", Label: "Status", Kind: adminstudio.FieldStatus, VisibleInList: true}},
		Actions:  []adminstudio.Action{{Label: "Articles", Href: "/admin/resources/article"}},
		Workflow: &adminstudio.Workflow{StateField: "status", States: []adminstudio.PublicationState{adminstudio.StateDraft, adminstudio.StatePublished}, Transitions: map[adminstudio.PublicationState][]adminstudio.PublicationState{adminstudio.StateDraft: {adminstudio.StatePublished}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler := adminstudio.GeneratedAdminHandler{Registry: registry, Options: adminstudio.GeneratedAdminOptions{
		API:             http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "csrf" },
		ValidateCSRF:    func(*http.Request) bool { return true },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
		ListRevisions: func(*http.Request, adminstudio.Collection, string, int) ([]adminstudio.RevisionSummary, error) {
			return nil, nil
		},
		GetRevision: func(*http.Request, adminstudio.Collection, string, int64) (adminstudio.RevisionSnapshot, error) {
			return adminstudio.RevisionSnapshot{}, nil
		},
		CanRestore:      func(*http.Request, adminstudio.Collection, string, int64) bool { return false },
		RestoreRevision: func(*http.Request, adminstudio.Collection, string, int64, int64) error { return nil },
		Locales: func(*http.Request, adminstudio.Collection) (adminstudio.LocaleOptions, error) {
			return adminstudio.LocaleOptions{Default: "en", Allowed: []string{"en", "fr"}}, nil
		},
		GetLocalized: func(*http.Request, adminstudio.Collection, string, string) (map[string]any, int64, error) {
			return nil, 0, nil
		},
		SaveLocalized: func(*http.Request, adminstudio.Collection, string, string, map[string]any, int64) error { return nil },
		ListLocalized: func(*http.Request, adminstudio.Collection, []string, string) (map[string]map[string]any, error) {
			return nil, nil
		},
		CanCopyLocale: func(*http.Request, adminstudio.Collection, string, string) bool { return true },
		CopyLocale:    func(*http.Request, adminstudio.Collection, string, string, string, int64) error { return nil },
		ListDraftPage: func(r *http.Request, _ adminstudio.Collection, locale string, offset, limit int) ([]adminstudio.DraftDocument, error) {
			rows, err := db.QueryContext(r.Context(), `SELECT resource_id,version,published_version,state,snapshot,updated_at FROM edition_drafts WHERE locale=? ORDER BY updated_at DESC,resource_id DESC LIMIT ? OFFSET ?`, locale, limit, offset)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var drafts []adminstudio.DraftDocument
			for rows.Next() {
				var draft adminstudio.DraftDocument
				var state, updated string
				var snapshot []byte
				if err := rows.Scan(&draft.ID, &draft.Version, &draft.PublishedVersion, &state, &snapshot, &updated); err != nil {
					return nil, err
				}
				draft.State, draft.Exists = adminstudio.PublicationState(state), true
				if err := json.Unmarshal(snapshot, &draft.Fields); err != nil {
					return nil, err
				}
				drafts = append(drafts, draft)
			}
			return drafts, rows.Err()
		},
		GetDraft: func(*http.Request, adminstudio.Collection, string, string) (adminstudio.DraftDocument, error) {
			return adminstudio.DraftDocument{}, nil
		},
		CanSaveDraft: func(*http.Request, adminstudio.Collection, string, string) bool { return true },
		SaveDraft: func(*http.Request, adminstudio.Collection, string, string, map[string]any, int64, int64) error {
			return nil
		},
		CanTransitionDraft: func(*http.Request, adminstudio.Collection, adminstudio.DraftDocument, adminstudio.PublicationState) bool {
			return true
		},
		TransitionDraft: func(*http.Request, adminstudio.Collection, string, string, adminstudio.PublicationState, int64, int64) error {
			return nil
		},
		CanDiscardDraft: func(*http.Request, adminstudio.Collection, adminstudio.DraftDocument) bool { return true },
		DiscardDraft:    func(*http.Request, adminstudio.Collection, string, string, int64) error { return nil },
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		url   string
		wants []string
		miss  []string
	}{
		{url: "/admin/resources/article/drafts?locale=fr", wants: []string{"Draft 27", "Draft 03", `name="locale" value="fr"`, "Next"}, miss: []string{"Draft 02", "Draft 01"}},
		{url: "/admin/resources/article/drafts?page=2&locale=fr", wants: []string{"Draft 02", "Draft 01", `name="locale" value="fr"`, "Previous"}, miss: []string{"Next"}},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, tc.url, nil)
		request.AddCookie(&http.Cookie{Name: "hydro_admin_locale", Value: "en"})
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", tc.url, response.Code, response.Body.String())
		}
		for _, want := range tc.wants {
			if !strings.Contains(response.Body.String(), want) {
				t.Errorf("%s missing %q: %s", tc.url, want, response.Body.String())
			}
		}
		for _, unwanted := range tc.miss {
			if strings.Contains(response.Body.String(), unwanted) {
				t.Errorf("%s unexpectedly contains %q", tc.url, unwanted)
			}
		}
	}
}
