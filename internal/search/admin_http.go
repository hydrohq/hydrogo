package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const adminSearchResultLimit = 8

type AdminSearchHandler struct {
	DB                 database.Executor
	Dialect            database.Dialect
	RequireTenantScope bool
	Modules            map[string]bool
}

type adminSearchItem struct {
	ID     string  `json:"id"`
	Locale string  `json:"-"`
	Score  float64 `json:"score"`
	Title  string  `json:"title"`
	Sub    string  `json:"sub"`
	Badge  string  `json:"badge"`
	Href   string  `json:"href"`
}

func (h AdminSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := database.RequireExecutor(h.DB); err != nil || (h.Dialect != database.SQLite && h.Dialect != database.Postgres) {
		http.Error(w, "search unavailable", http.StatusServiceUnavailable)
		return
	}
	text := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(text) < 2 || len(text) > MaxQueryLength {
		writeAdminSearchError(w, http.StatusBadRequest, "search query must contain between 2 and 256 characters")
		return
	}
	query := Query{Text: text, Locale: r.URL.Query().Get("locale"), Limit: adminSearchResultLimit}
	if err := query.Validate(); err != nil {
		writeAdminSearchError(w, http.StatusBadRequest, "invalid search query")
		return
	}
	query.Locale = canonical.Locale(query.Locale)
	tenantID := ""
	if h.RequireTenantScope {
		scope, err := tenant.Require(r.Context())
		if err != nil {
			writeAdminSearchError(w, http.StatusForbidden, "search unavailable")
			return
		}
		tenantID = scope.TenantID
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		writeAdminSearchError(w, http.StatusForbidden, "search unavailable")
		return
	}
	items, err := h.search(r.Context(), query, tenantID, user.Role == auth.RoleAdmin, user.Role == auth.RoleAdmin)
	if err != nil {
		writeAdminSearchError(w, http.StatusServiceUnavailable, "search unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = json.NewEncoder(w).Encode(struct {
		Items []adminSearchItem `json:"items"`
		Total int               `json:"total"`
	}{Items: items, Total: len(items)})
}

func (h AdminSearchHandler) search(ctx context.Context, query Query, tenantID string, includeMembers, includeSettings bool) ([]adminSearchItem, error) {
	documents := make([]adminSearchItem, 0, 64)
	for _, source := range []adminSearchSource{
		{"pages", "pages", "id", "locale", "title", "body", "Sayfa"},
		{"news", "news_items", "id", "locale", "title", "excerpt", "Haber"},
		{"members", "members", "id", "", "email", "", "Üye"},
		{"media", "media_assets", "id", "", "original_filename", "mime_type", "Medya"},
	} {
		if !h.Modules[source.module] || (source.table == "members" && !includeMembers) {
			continue
		}
		items, err := h.readSource(ctx, source, tenantID, query)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		documents = append(documents, items...)
	}
	if includeSettings && h.Modules["site"] {
		items, err := h.readSettings(ctx, tenantID, query.Text)
		if err != nil {
			return nil, err
		}
		documents = append(documents, items...)
	}
	terms := tokenize(strings.ToLower(strings.TrimSpace(query.Text)))
	items := make([]adminSearchItem, 0, len(documents))
	for _, item := range documents {
		item.Score = adminSearchScore(item.Title, item.Sub, terms)
		if item.Score == 0 {
			continue
		}
		item.Sub = snippet(item.Sub, terms)
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].ID < items[j].ID
	})
	if len(items) > adminSearchResultLimit {
		items = items[:adminSearchResultLimit]
	}
	return items, nil
}

func adminSearchScore(title, body string, terms []string) float64 {
	title, body = strings.ToLower(title), strings.ToLower(body)
	score := 0.0
	for _, term := range terms {
		if strings.Contains(title, term) {
			score += 2
		}
		if strings.Contains(body, term) {
			score++
		}
	}
	return score
}

type adminSearchSource struct {
	module, table, id, locale, title, body, label string
}

func (h AdminSearchHandler) readSource(ctx context.Context, source adminSearchSource, tenantID string, searchQuery Query) ([]adminSearchItem, error) {
	query := "SELECT " + source.id + ","
	if source.locale == "" {
		query += "''"
	} else {
		query += source.locale
	}
	query += "," + source.title + ","
	if source.body == "" {
		query += "''"
	} else {
		query += source.body
	}
	query += " FROM " + source.table + " WHERE 1=1"
	args := []any{}
	if source.table == "pages" || source.table == "news_items" {
		query += " AND deleted_at IS NULL"
		if source.table == "news_items" {
			query += " AND status!='retracted'"
		}
		if searchQuery.Locale != "" {
			query += " AND locale=" + h.placeholder(len(args)+1)
			args = append(args, searchQuery.Locale)
		}
	} else if source.table == "members" {
		query += " AND active="
		if h.Dialect == database.Postgres {
			query += "TRUE"
		} else {
			query += "1"
		}
	} else if source.table == "media_assets" {
		query += " AND deleted_at IS NULL AND visibility='public' AND status='ready'"
	}
	if tenantID != "" || h.RequireTenantScope {
		query += " AND tenant_id=" + h.placeholder(len(args)+1)
		args = append(args, tenantID)
	}
	matchExpression := "COALESCE(" + source.title + ",'')"
	if source.body != "" {
		matchExpression += " || ' ' || COALESCE(" + source.body + ",'')"
	}
	conditions, terms := h.likeTerms(matchExpression, searchQuery.Text)
	if len(conditions) > 0 {
		query += " AND (" + strings.Join(conditions, " OR ") + ")"
		args = append(args, terms...)
	}
	query += " ORDER BY " + source.id + " LIMIT 500"
	rows, err := h.query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items := make([]adminSearchItem, 0, len(rows))
	for _, row := range rows {
		href := adminSearchHref(source.table, row.id)
		if source.locale != "" {
			href += "?locale=" + url.QueryEscape(row.locale)
		}
		items = append(items, adminSearchItem{ID: source.table + ":" + row.id, Locale: row.locale, Title: row.title, Sub: row.body, Badge: source.label, Href: href})
	}
	return items, nil
}

func (h AdminSearchHandler) readSettings(ctx context.Context, tenantID, searchText string) ([]adminSearchItem, error) {
	query := "SELECT key,'',key,'' FROM site_settings WHERE 1=1"
	args := []any{}
	if tenantID != "" || h.RequireTenantScope {
		query += " AND tenant_id=" + h.placeholder(1)
		args = append(args, tenantID)
	}
	conditions, terms := h.likeTerms("COALESCE(key,'')", searchText)
	if len(conditions) > 0 {
		query += " AND (" + strings.Join(conditions, " OR ") + ")"
		args = append(args, terms...)
	}
	query += " ORDER BY key LIMIT 200"
	rows, err := h.query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items := make([]adminSearchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminSearchItem{ID: "setting:" + row.title, Title: row.title, Badge: "Ayar", Href: "/admin/site"})
	}
	return items, nil
}

func (h AdminSearchHandler) likeTerms(expression, searchText string) ([]string, []any) {
	terms := tokenize(strings.ToLower(searchText))
	conditions := make([]string, 0, len(terms))
	args := make([]any, 0, len(terms))
	for _, term := range terms {
		conditions = append(conditions, "LOWER("+expression+") LIKE "+h.placeholder(len(args)+1)+" ESCAPE '!'")
		args = append(args, "%"+strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(term)+"%")
	}
	return conditions, args
}

type adminSearchRow struct{ id, locale, title, body string }

func (h AdminSearchHandler) query(ctx context.Context, statement string, args ...any) ([]adminSearchRow, error) {
	queryer, ok := h.DB.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("search database cannot query")
	}
	rows, err := queryer.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]adminSearchRow, 0, 64)
	for rows.Next() {
		var item adminSearchRow
		if err := rows.Scan(&item.id, &item.locale, &item.title, &item.body); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (h AdminSearchHandler) placeholder(index int) string {
	return sqlPlaceholder(h.Dialect, index)
}

func adminSearchHref(table, id string) string {
	path := map[string]string{"pages": "/admin/pages/", "news_items": "/admin/news/", "members": "/admin/members/", "media_assets": "/admin/media"}[table]
	if table == "media_assets" {
		return path + "?q=" + url.QueryEscape(id)
	}
	return path + url.PathEscape(id)
}

func writeAdminSearchError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
