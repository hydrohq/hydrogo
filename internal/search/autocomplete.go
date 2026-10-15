package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type AutocompleteHandler struct {
	DB                 database.QueryExecutor
	Dialect            database.Dialect
	RequireTenantScope bool
	Modules            map[string]bool
}
type publicSuggestion struct{ id, locale, title, slug, kind string }

// Suggestion is one public autocomplete hit. Kind is "pages" or "news"; the
// frontend builds its own URL from locale and slug.
type Suggestion struct {
	Title  string `json:"title"`
	Kind   string `json:"kind"`
	Locale string `json:"locale"`
	Slug   string `json:"slug"`
}

func (h AutocompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 || len(query) > 80 {
		http.Error(w, "invalid search query", http.StatusBadRequest)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" {
		locale = "tr"
	}
	tenantID := ""
	if h.RequireTenantScope {
		scope, err := tenant.Require(r.Context())
		if err != nil {
			http.Error(w, "search unavailable", http.StatusForbidden)
			return
		}
		tenantID = scope.TenantID
	}
	items, err := h.suggest(r.Context(), query, locale, tenantID)
	if err != nil {
		http.Error(w, "search unavailable", http.StatusServiceUnavailable)
		return
	}
	suggestions := make([]Suggestion, 0, len(items))
	for _, item := range items {
		suggestions = append(suggestions, Suggestion{Title: item.title, Kind: item.kind, Locale: item.locale, Slug: item.slug})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=30")
	if h.RequireTenantScope {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"items": suggestions})
}

func (h AutocompleteHandler) suggest(ctx context.Context, term, locale, tenantID string) ([]publicSuggestion, error) {
	if h.RequireTenantScope && tenantID == "" {
		return nil, errors.New("missing tenant scope")
	}
	if err := database.RequireExecutor(h.DB); err != nil {
		return nil, err
	}
	if h.Dialect != database.SQLite && h.Dialect != database.Postgres {
		return nil, errors.New("unsupported search database")
	}
	pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(term) + "%"
	out := make([]publicSuggestion, 0, 5)
	for _, source := range []struct {
		enabled     bool
		kind, table string
	}{{h.Modules["pages"], "pages", "pages"}, {h.Modules["news"], "news", "news_items"}} {
		if !source.enabled {
			continue
		}
		if len(out) >= 5 {
			break
		}
		args := []any{locale, pattern}
		stmt := "SELECT id,locale,title,slug FROM " + source.table + " WHERE status='published' AND deleted_at IS NULL AND locale=" + h.ph(1) + " AND title LIKE " + h.ph(2) + " ESCAPE '\\'"
		if source.table == "news_items" {
			stmt += " AND status='published'"
		}
		if tenantID != "" || h.RequireTenantScope {
			stmt += " AND tenant_id=" + h.ph(3)
			args = append(args, tenantID)
		}
		stmt += " ORDER BY title,id LIMIT " + h.ph(len(args)+1)
		args = append(args, 5-len(out))
		rows, err := h.DB.QueryContext(ctx, stmt, args...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		for rows.Next() {
			var row publicSuggestion
			row.kind = source.kind
			if err := rows.Scan(&row.id, &row.locale, &row.title, &row.slug); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, row)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (h AutocompleteHandler) ph(i int) string {
	if h.Dialect == database.Postgres {
		return "$" + strconv.Itoa(i)
	}
	return "?"
}
