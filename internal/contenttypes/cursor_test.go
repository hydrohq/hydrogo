package contenttypes

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

func TestCursorPagingWalksEveryRecordOnce(t *testing.T) {
	service, types := newServiceFor(t, searchableFAQ)
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		if _, err := service.CreateRecord(ctx, "faq", map[string]any{"question": "Q" + strconv.Itoa(i), "category": "general", "sort_order": float64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	public := publicMux(PublicAPI{Service: service, Types: types})
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})

	walk := func(mux http.Handler, base, sortParam string) []string {
		t.Helper()
		var seen []string
		after := ""
		for page := 0; page < 10; page++ {
			target := base + "?limit=3" + sortParam
			if after != "" {
				target += "&after=" + url.QueryEscape(after)
			}
			recorder, body := call(t, mux, http.MethodGet, target, "")
			if recorder.Code != http.StatusOK {
				t.Fatalf("%s: %d %s", target, recorder.Code, recorder.Body)
			}
			for _, item := range body["data"].([]any) {
				seen = append(seen, item.(map[string]any)["question"].(string))
			}
			meta := body["metadata"].(map[string]any)
			next, _ := meta["next_cursor"].(string)
			if meta["has_more"] != true {
				if next != "" {
					t.Fatalf("last page carries a cursor: %v", meta)
				}
				return seen
			}
			if next == "" {
				t.Fatalf("%s: has_more without next_cursor: %v", target, meta)
			}
			after = next
		}
		t.Fatal("paging did not end")
		return nil
	}
	for name, test := range map[string]struct {
		mux  http.Handler
		base string
		sort string
	}{
		"public default":      {public, "/api/v1/content/faq", ""},
		"public created":      {public, "/api/v1/content/faq", "&sort=created_at"},
		"public created desc": {public, "/api/v1/content/faq", "&sort=-created_at"},
		"admin default":       {admin, AdminAPIPath + "/faq", ""},
	} {
		t.Run(name, func(t *testing.T) {
			seen := walk(test.mux, test.base, test.sort)
			unique := map[string]bool{}
			for _, question := range seen {
				unique[question] = true
			}
			if len(seen) != 7 || len(unique) != 7 {
				t.Fatalf("saw %d records (%d distinct): %v", len(seen), len(unique), seen)
			}
		})
	}

	// A field sort cannot continue by cursor: no cursor, and after is refused.
	_, body := call(t, public, http.MethodGet, "/api/v1/content/faq?limit=3&sort=sort_order", "")
	if cursor, present := body["metadata"].(map[string]any)["next_cursor"]; present {
		t.Fatalf("field sort offers a cursor: %v", cursor)
	}
	_, body = call(t, public, http.MethodGet, "/api/v1/content/faq?limit=3", "")
	cursor := body["metadata"].(map[string]any)["next_cursor"].(string)
	for _, target := range []string{
		"/api/v1/content/faq?after=" + url.QueryEscape(cursor) + "&offset=3",
		"/api/v1/content/faq?after=" + url.QueryEscape(cursor) + "&sort=-created_at",
		"/api/v1/content/faq?after=" + url.QueryEscape(cursor) + "&sort=sort_order",
		"/api/v1/content/faq?after=not-a-cursor",
		"/api/v1/content/faq?after=" + url.QueryEscape(cursor) + "&after=" + url.QueryEscape(cursor),
	} {
		if recorder, _ := call(t, public, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", target, recorder.Code)
		}
	}
}
