// Package sectoral contains the small shared persistence contract used by
// optional, content-oriented modules. It deliberately starts no workers or
// routes and applies no migrations.
package sectoral

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var (
	ErrInvalid       = errors.New("invalid sectoral content")
	ErrNotFound      = errors.New("sectoral content not found")
	ErrConflict      = errors.New("sectoral content changed by another editor")
	ErrAuditRequired = errors.New("sectoral audit writer is required")
)

type Base struct {
	ID        string   `json:"id"`
	Locale    string   `json:"locale"`
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary,omitempty"`
	Body      string   `json:"body,omitempty"`
	Status    string   `json:"status"`
	Featured  bool     `json:"featured"`
	SortOrder int      `json:"sort_order"`
	Tags      []string `json:"tags,omitempty"`
	Version   int      `json:"version"`
}

func (b Base) SectoralBase() Base { return b }

func ValidateBase(b *Base) error {
	if b == nil {
		return ErrInvalid
	}
	b.ID = strings.TrimSpace(b.ID)
	b.Locale = canonical.Locale(b.Locale)
	b.Title = strings.TrimSpace(b.Title)
	b.Tags = NormalizeTags(b.Tags, 50)
	slug, err := canonical.Path(b.Slug)
	if err != nil || slug == "" {
		return ErrInvalid
	}
	b.Slug = slug
	if b.ID == "" || len(b.ID) > 128 || b.Locale == "" || len(b.Locale) > 16 || b.Slug == "" || len(b.Slug) > 256 || b.Title == "" || len(b.Title) > 256 || len(b.Summary) > 4000 || len(b.Body) > 1<<20 || b.SortOrder < -100000 || b.SortOrder > 100000 {
		return ErrInvalid
	}
	if b.Status == "" {
		b.Status = "draft"
	}
	if b.Version == 0 {
		b.Version = 1
	}
	if b.Version < 1 {
		return ErrInvalid
	}
	if b.Status != "draft" && b.Status != "published" && b.Status != "archived" {
		return ErrInvalid
	}
	return nil
}

type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func CloneBase(source Base, id, slug string) Base {
	source.ID = strings.TrimSpace(id)
	source.Slug = strings.TrimSpace(slug)
	source.Title = strings.TrimSpace(source.Title) + " (kopya)"
	source.Status, source.Featured = "draft", false
	source.Version = 0
	return source
}

func NormalizeTags(values []string, limit int) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, min(len(values), limit))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 64 {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
		if len(result) == limit {
			break
		}
	}
	return result
}

func ValidWebURL(value string, optional bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return optional
	}
	if len(value) > 2048 || strings.ContainsAny(value, "\\\r\n\x00") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}
	if u.IsAbs() {
		return (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
	}
	return u.Host == "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}

func SanitizeWebURL(value string) string {
	value = strings.TrimSpace(value)
	if !ValidWebURL(value, false) {
		return ""
	}
	return value
}

func SafeSlugSuffix(base, suffix string) string {
	base, suffix = strings.TrimSpace(base), strings.TrimSpace(suffix)
	if base == "" || suffix == "" {
		return ""
	}
	return base + "-" + suffix
}

func ValidEmail(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value && len(value) <= 320
}

func ValidLinks(values []Link, limit int) bool {
	if len(values) > limit {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value.Label) == "" || len(value.Label) > 80 || !ValidWebURL(value.URL, false) {
			return false
		}
	}
	return true
}

func ValidTimezone(value string) bool {
	_, err := time.LoadLocation(strings.TrimSpace(value))
	return err == nil
}

func SafeLabel(value string) string {
	value = strings.TrimSpace(value)
	for _, r := range value {
		if unicode.IsControl(r) {
			return ""
		}
	}
	return value
}

type Entity interface{ SectoralBase() Base }

type AuditWriter interface {
	Record(context.Context, database.Executor, string, string, string) error
}

func RecordAudit(ctx context.Context, tx database.Executor, writer AuditWriter, module, action, entityID string, version int) error {
	if writer == nil {
		return ErrAuditRequired
	}
	digest := sha256.Sum256([]byte(module + "\x00" + action + "\x00" + entityID + "\x00" + fmt.Sprint(version)))
	return writer.Record(ctx, tx, "sectoral-"+hex.EncodeToString(digest[:]), module+"."+action, entityID)
}

type Repository[T Entity] interface {
	Create(context.Context, database.Executor, T) error
	Update(context.Context, database.Executor, T) error
	Get(context.Context, database.Executor, string) (T, error)
	List(context.Context, database.QueryExecutor, int) ([]T, error)
	ListPublished(context.Context, database.QueryExecutor, string, int) ([]T, error)
	Delete(context.Context, database.Executor, string) error
}

type Store[T Entity] struct {
	Dialect database.Dialect
	Table   string
}

var tableName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func (s Store[T]) validate(db database.Executor) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if s.Dialect != database.SQLite && s.Dialect != database.Postgres && s.Dialect != database.MySQL {
		return errors.New("unsupported database dialect")
	}
	if !tableName.MatchString(s.Table) {
		return errors.New("invalid sectoral table")
	}
	return nil
}

func (s Store[T]) bind(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func scopeID(ctx context.Context) string {
	if scope, err := tenant.FromContext(ctx); err == nil {
		return scope.TenantID
	}
	return ""
}

// TenantID returns the active tenant discriminator, or the empty single-tenant scope.
func TenantID(ctx context.Context) string { return scopeID(ctx) }

func encode[T Entity](item T) (Base, []byte, error) {
	base := item.SectoralBase()
	if err := ValidateBase(&base); err != nil {
		return Base{}, nil, err
	}
	data, err := json.Marshal(item)
	if err != nil {
		return Base{}, nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return Base{}, nil, err
	}
	for key, value := range map[string]any{"id": base.ID, "locale": base.Locale, "slug": base.Slug, "title": base.Title, "status": base.Status, "featured": base.Featured, "sort_order": base.SortOrder, "tags": base.Tags, "version": base.Version} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return Base{}, nil, err
		}
		payload[key] = encoded
	}
	data, err = json.Marshal(payload)
	if err != nil {
		return Base{}, nil, err
	}
	if len(data) > 1<<20 {
		return Base{}, nil, ErrInvalid
	}
	return base, data, nil
}

func (s Store[T]) Create(ctx context.Context, db database.Executor, item T) error {
	if err := s.validate(db); err != nil {
		return err
	}
	base, data, err := encode(item)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	args := []any{base.ID, scopeID(ctx), base.Locale, base.Slug, base.Title, base.Status, boolArg(s.Dialect, base.Featured), base.SortOrder, base.Version, string(data), now}
	_, err = db.ExecContext(ctx, "INSERT INTO "+s.Table+" (id,tenant_id,locale,slug,title,status,featured,sort_order,version,payload,updated_at) VALUES ("+s.bind(1)+","+s.bind(2)+","+s.bind(3)+","+s.bind(4)+","+s.bind(5)+","+s.bind(6)+","+s.bind(7)+","+s.bind(8)+","+s.bind(9)+","+s.bind(10)+","+s.bind(11)+")", args...)
	return err
}

func (s Store[T]) Update(ctx context.Context, db database.Executor, item T) error {
	if err := s.validate(db); err != nil {
		return err
	}
	base, data, err := encode(item)
	if err != nil {
		return err
	}
	if base.Version < 2 {
		return ErrInvalid
	}
	args := []any{base.Locale, base.Slug, base.Title, base.Status, boolArg(s.Dialect, base.Featured), base.SortOrder, base.Version, string(data), time.Now().UTC(), base.ID, scopeID(ctx), base.Version - 1}
	result, err := db.ExecContext(ctx, "UPDATE "+s.Table+" SET locale="+s.bind(1)+",slug="+s.bind(2)+",title="+s.bind(3)+",status="+s.bind(4)+",featured="+s.bind(5)+",sort_order="+s.bind(6)+",version="+s.bind(7)+",payload="+s.bind(8)+",updated_at="+s.bind(9)+" WHERE id="+s.bind(10)+" AND tenant_id="+s.bind(11)+" AND version="+s.bind(12), args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrConflict
	}
	return nil
}

func (s Store[T]) Get(ctx context.Context, db database.Executor, id string) (T, error) {
	var zero T
	if err := s.validate(db); err != nil {
		return zero, err
	}
	var data string
	if err := db.QueryRowContext(ctx, "SELECT payload FROM "+s.Table+" WHERE id="+s.bind(1)+" AND tenant_id="+s.bind(2), id, scopeID(ctx)).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero, ErrNotFound
		}
		return zero, err
	}
	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return zero, err
	}
	return item, nil
}

func (s Store[T]) List(ctx context.Context, db database.QueryExecutor, limit int) ([]T, error) {
	return s.list(ctx, db, "", "", limit)
}

func (s Store[T]) ListPublished(ctx context.Context, db database.QueryExecutor, locale string, limit int) ([]T, error) {
	return s.list(ctx, db, "published", canonical.Locale(locale), limit)
}

func (s Store[T]) list(ctx context.Context, db database.QueryExecutor, status, locale string, limit int) ([]T, error) {
	if err := s.validate(db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	query := "SELECT payload FROM " + s.Table + " WHERE tenant_id=" + s.bind(1)
	args := []any{scopeID(ctx)}
	if status != "" {
		args = append(args, status)
		query += " AND status=" + s.bind(len(args))
	}
	if locale != "" {
		args = append(args, locale)
		query += " AND locale=" + s.bind(len(args))
	}
	args = append(args, limit)
	query += " ORDER BY featured DESC,sort_order ASC,updated_at DESC,id ASC LIMIT " + s.bind(len(args))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]T, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store[T]) Delete(ctx context.Context, db database.Executor, id string) error {
	if err := s.validate(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, "DELETE FROM "+s.Table+" WHERE id="+s.bind(1)+" AND tenant_id="+s.bind(2), id, scopeID(ctx))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func boolArg(d database.Dialect, value bool) any {
	if d == database.Postgres {
		return value
	}
	if value {
		return 1
	}
	return 0
}
