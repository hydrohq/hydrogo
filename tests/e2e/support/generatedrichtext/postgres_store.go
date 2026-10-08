package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type postgresEditorialStore struct {
	db      *sql.DB
	adminDB *sql.DB
	schema  string
}

func openPostgresEditorialStore(dsn string) (*postgresEditorialStore, error) {
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		return nil, err
	}
	schema := fmt.Sprintf("generated_richtext_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = adminDB.Close()
		return nil, err
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		_, _ = adminDB.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = adminDB.Close()
		return nil, err
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		_, _ = adminDB.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = adminDB.Close()
		return nil, err
	}
	store := &postgresEditorialStore{db: db, adminDB: adminDB, schema: schema}
	if err := store.initialize(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *postgresEditorialStore) Close() error {
	_, dropErr := s.adminDB.ExecContext(context.Background(), "DROP SCHEMA "+s.schema+" CASCADE")
	dbErr := s.db.Close()
	adminErr := s.adminDB.Close()
	return errors.Join(dropErr, dbErr, adminErr)
}

func (s *postgresEditorialStore) initialize(ctx context.Context) error {
	for _, statement := range []string{
		`CREATE TABLE stories (id TEXT PRIMARY KEY, title TEXT NOT NULL, status TEXT NOT NULL, version BIGINT NOT NULL)`,
		`CREATE TABLE stories_drafts (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version BIGINT NOT NULL, published_version BIGINT NOT NULL, state TEXT NOT NULL, snapshot JSONB NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY(resource_id, locale))`,
		`CREATE TABLE stories_locales (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version BIGINT NOT NULL, values_json JSONB NOT NULL, PRIMARY KEY(resource_id, locale))`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO stories(id,title,status,version) VALUES($1,$2,$3,$4)`, "story-1", "Published story", "published", 1)
	if err != nil {
		return err
	}
	for _, seed := range []struct {
		id, locale, state  string
		version, published int64
		fields             map[string]any
	}{
		{"story-1", "en", "review", 1, 1, map[string]any{"title": "Review copy"}},
		{"story-1", "tr", "draft", 1, 1, map[string]any{"title": "Türkçe taslak"}},
		{"story-discard", "en", "draft", 1, 0, map[string]any{"title": "Discard-only draft"}},
	} {
		encoded, err := json.Marshal(seed.fields)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO stories_drafts(resource_id,locale,version,published_version,state,snapshot,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, seed.id, seed.locale, seed.version, seed.published, seed.state, encoded, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO stories_locales(resource_id,locale,version,values_json) VALUES($1,$2,$3,$4)`, "story-1", "tr", 1, []byte(`{"title":"Yayımlanmış Türkçe metin"}`))
	return err
}

func (s *postgresEditorialStore) Published(id string) (map[string]any, error) {
	var title, state string
	var version int64
	err := s.db.QueryRow(`SELECT title,status,version FROM stories WHERE id=$1`, id).Scan(&title, &state, &version)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "title": title, "status": state, "version": version}, nil
}

func (s *postgresEditorialStore) PublishedList() ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id,title,status,version FROM stories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var id, title, state string
		var version int64
		if err := rows.Scan(&id, &title, &state, &version); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"id": id, "title": title, "status": state, "version": version})
	}
	return result, rows.Err()
}

func (s *postgresEditorialStore) Drafts(locale string, limit int) ([]adminstudio.DraftDocument, error) {
	rows, err := s.db.Query(`SELECT resource_id,version,published_version,state,snapshot,updated_at FROM stories_drafts WHERE locale=$1 ORDER BY updated_at DESC LIMIT $2`, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []adminstudio.DraftDocument
	for rows.Next() {
		item, err := scanDraft(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *postgresEditorialStore) Draft(id, locale string) (adminstudio.DraftDocument, error) {
	item, err := scanDraft(s.db.QueryRow(`SELECT resource_id,version,published_version,state,snapshot,updated_at FROM stories_drafts WHERE resource_id=$1 AND locale=$2`, id, locale))
	if !errors.Is(err, sql.ErrNoRows) {
		return item, err
	}
	item.ID = id
	err = s.db.QueryRow(`SELECT version FROM stories WHERE id=$1`, id).Scan(&item.PublishedVersion)
	return item, err
}

func scanDraft(row interface{ Scan(...any) error }) (adminstudio.DraftDocument, error) {
	var item adminstudio.DraftDocument
	var state, updated string
	var snapshot []byte
	err := row.Scan(&item.ID, &item.Version, &item.PublishedVersion, &state, &snapshot, &updated)
	if err != nil {
		return adminstudio.DraftDocument{}, err
	}
	if err := json.Unmarshal(snapshot, &item.Fields); err != nil {
		return adminstudio.DraftDocument{}, err
	}
	item.State = adminstudio.PublicationState(state)
	item.Exists = true
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return item, nil
}

func (s *postgresEditorialStore) SaveDraft(id, locale string, fields map[string]any, expectedDraft, expectedPublished int64) error {
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var published int64
	if err := tx.QueryRow(`SELECT version FROM stories WHERE id=$1`, id).Scan(&published); err != nil || published != expectedPublished {
		return fmt.Errorf("published version conflict")
	}
	var current int64
	err = tx.QueryRow(`SELECT version FROM stories_drafts WHERE resource_id=$1 AND locale=$2`, id, locale).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		if expectedDraft != 0 {
			return fmt.Errorf("draft version conflict")
		}
	} else if err != nil || current != expectedDraft {
		return fmt.Errorf("draft version conflict")
	}
	_, err = tx.Exec(`INSERT INTO stories_drafts(resource_id,locale,version,published_version,state,snapshot,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(resource_id,locale) DO UPDATE SET version=EXCLUDED.version,published_version=EXCLUDED.published_version,state=EXCLUDED.state,snapshot=EXCLUDED.snapshot,updated_at=EXCLUDED.updated_at`, id, locale, expectedDraft+1, expectedPublished, string(adminstudio.StateDraft), encoded, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *postgresEditorialStore) PublishDraft(id, locale string, expectedDraft, expectedPublished int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var draftVersion, publishedVersion int64
	var snapshot []byte
	if err := tx.QueryRow(`SELECT version,published_version,snapshot FROM stories_drafts WHERE resource_id=$1 AND locale=$2 FOR UPDATE`, id, locale).Scan(&draftVersion, &publishedVersion, &snapshot); err != nil || draftVersion != expectedDraft || publishedVersion != expectedPublished {
		return fmt.Errorf("draft version conflict")
	}
	var published int64
	if err := tx.QueryRow(`SELECT version FROM stories WHERE id=$1 FOR UPDATE`, id).Scan(&published); err != nil || published != expectedPublished {
		return fmt.Errorf("published version conflict")
	}
	var fields map[string]any
	if err := json.Unmarshal(snapshot, &fields); err != nil {
		return err
	}
	if locale == "tr" {
		localized, err := json.Marshal(map[string]any{"title": fields["title"]})
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO stories_locales(resource_id,locale,version,values_json) VALUES($1,$2,2,$3) ON CONFLICT(resource_id,locale) DO UPDATE SET version=stories_locales.version+1,values_json=EXCLUDED.values_json`, id, locale, localized)
		if err != nil {
			return err
		}
	} else {
		result, err := tx.Exec(`UPDATE stories SET title=$1,status='published',version=version+1 WHERE id=$2 AND version=$3`, fields["title"], id, expectedPublished)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return fmt.Errorf("published version conflict")
		}
	}
	if _, err := tx.Exec(`DELETE FROM stories_drafts WHERE resource_id=$1 AND locale=$2 AND version=$3`, id, locale, expectedDraft); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *postgresEditorialStore) DiscardDraft(id, locale string, expected int64) error {
	result, err := s.db.Exec(`DELETE FROM stories_drafts WHERE resource_id=$1 AND locale=$2 AND version=$3`, id, locale, expected)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return fmt.Errorf("draft version conflict")
	}
	return nil
}

func (s *postgresEditorialStore) Localized(id, locale string) (map[string]any, int64, error) {
	var version int64
	var values []byte
	err := s.db.QueryRow(`SELECT version,values_json FROM stories_locales WHERE resource_id=$1 AND locale=$2`, id, locale).Scan(&version, &values)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]any{}, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var result map[string]any
	err = json.Unmarshal(values, &result)
	return result, version, err
}

func (s *postgresEditorialStore) SaveLocalized(id, locale string, fields map[string]any, expected int64) error {
	values, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int64
	err = tx.QueryRow(`SELECT version FROM stories_locales WHERE resource_id=$1 AND locale=$2`, id, locale).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		if expected != 0 {
			return fmt.Errorf("locale version conflict")
		}
	} else if err != nil || version != expected {
		return fmt.Errorf("locale version conflict")
	}
	_, err = tx.Exec(`INSERT INTO stories_locales(resource_id,locale,version,values_json) VALUES($1,$2,$3,$4) ON CONFLICT(resource_id,locale) DO UPDATE SET version=EXCLUDED.version,values_json=EXCLUDED.values_json`, id, locale, expected+1, values)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *postgresEditorialStore) CopyLocale(id, from, to string, expected int64) error {
	var title string
	if err := s.db.QueryRow(`SELECT title FROM stories WHERE id=$1`, id).Scan(&title); err != nil {
		return err
	}
	var current int64
	err := s.db.QueryRow(`SELECT version FROM stories_locales WHERE resource_id=$1 AND locale=$2`, id, to).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		if expected != 0 {
			return fmt.Errorf("locale version conflict")
		}
	} else if err != nil || current != expected {
		return fmt.Errorf("locale version conflict")
	}
	values, _ := json.Marshal(map[string]any{"title": title})
	_, err = s.db.Exec(`INSERT INTO stories_locales(resource_id,locale,version,values_json) VALUES($1,$2,$3,$4) ON CONFLICT(resource_id,locale) DO UPDATE SET version=EXCLUDED.version,values_json=EXCLUDED.values_json`, id, to, expected+1, values)
	return err
}
