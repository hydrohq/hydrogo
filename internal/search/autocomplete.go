package search

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

type AutocompleteHandler struct {
	DB                 database.QueryExecutor
	Dialect            database.Dialect
	RequireTenantScope bool
	Modules            map[string]bool
}
type publicSuggestion struct{ id, locale, title, slug, kind string }

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
	suggestions := make([]ui.SearchSuggestion, 0, len(items))
	for _, item := range items {
		base := "/public/pages/"
		if item.kind == "news" {
			base = "/public/news/"
		}
		suggestions = append(suggestions, ui.SearchSuggestion{Title: item.title, Category: map[string]string{"pages": "Page", "news": "News"}[item.kind], URL: base + url.PathEscape(item.locale) + "/" + url.PathEscape(item.slug) + "/html"})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=30")
	if h.RequireTenantScope {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	if err := ui.SearchSuggestions(suggestions).Render(r.Context(), w); err != nil {
		return
	}
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
