package tenant

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	ErrInvalidSnapshot = errors.New("invalid tenant snapshot archive")
	ErrTenantNotFound  = errors.New("tenant not found")
)

// SnapshotManifest describes the metadata and inventory of an exported tenant.
type SnapshotManifest struct {
	FormatVersion int              `json:"format_version"`
	TenantID      string           `json:"tenant_id"`
	ExportedAt    time.Time        `json:"exported_at"`
	TableCounts   map[string]int64 `json:"table_counts"`
	TotalRows     int64            `json:"total_rows"`
	Checksum      string           `json:"checksum"`
}

// TableExport represents exported rows and columns of a specific database table.
type TableExport struct {
	Name    string           `json:"name"`
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

// ExportSnapshot writes a compressed tar.gz archive of the tenant's complete state to w.
func ExportSnapshot(ctx context.Context, db SQLExecutor, dialect string, tenantID string, w io.Writer) (*SnapshotManifest, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}

	p1 := "?"
	if dialect == "postgres" {
		p1 = "$1"
	}

	// Verify tenant exists
	var tenantExists int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM tenants WHERE tenant_id = "+p1, tenantID).Scan(&tenantExists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) && !isMissingTableError(err) {
		return nil, fmt.Errorf("verify tenant %s: %w", tenantID, err)
	}

	manifest := &SnapshotManifest{
		FormatVersion: 1,
		TenantID:      tenantID,
		ExportedAt:    time.Now().UTC(),
		TableCounts:   make(map[string]int64),
	}

	// Prepare tables to export
	allTables := append([]string{"tenants"}, BackfillTables...)

	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	hasher := sha256.New()
	var totalRows int64

	for _, table := range allTables {
		query := fmt.Sprintf("SELECT * FROM %s WHERE tenant_id = %s", table, p1) // #nosec G201 -- table is verified from fixed whitelist
		rows, err := db.QueryContext(ctx, query, tenantID)
		if err != nil {
			if isMissingTableError(err) {
				continue
			}
			return nil, fmt.Errorf("query table %s: %w", table, err)
		}

		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}

		var tableRows []map[string]any
		for rows.Next() {
			values := make([]any, len(cols))
			valuePtrs := make([]any, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				rows.Close()
				return nil, err
			}

			rowMap := make(map[string]any, len(cols))
			for i, col := range cols {
				val := values[i]
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
				rowMap[col] = val
			}
			tableRows = append(tableRows, rowMap)
		}
		rows.Close()

		if len(tableRows) > 0 {
			count := int64(len(tableRows))
			manifest.TableCounts[table] = count
			totalRows += count

			exportData := TableExport{
				Name:    table,
				Columns: cols,
				Rows:    tableRows,
			}

			jsonData, err := json.Marshal(exportData)
			if err != nil {
				return nil, err
			}

			hasher.Write(jsonData)

			hdr := &tar.Header{
				Name:    fmt.Sprintf("tables/%s.json", table),
				Mode:    0o600,
				Size:    int64(len(jsonData)),
				ModTime: manifest.ExportedAt,
			}
			if err := tarWriter.WriteHeader(hdr); err != nil {
				return nil, err
			}
			if _, err := tarWriter.Write(jsonData); err != nil {
				return nil, err
			}
		}
	}

	manifest.TotalRows = totalRows
	manifest.Checksum = hex.EncodeToString(hasher.Sum(nil))

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}

	manifestHdr := &tar.Header{
		Name:    "manifest.json",
		Mode:    0o600,
		Size:    int64(len(manifestJSON)),
		ModTime: manifest.ExportedAt,
	}
	if err := tarWriter.WriteHeader(manifestHdr); err != nil {
		return nil, err
	}
	if _, err := tarWriter.Write(manifestJSON); err != nil {
		return nil, err
	}

	return manifest, nil
}

// ImportSnapshot restores an exported tenant snapshot archive from r, re-mapping to targetTenantID if provided.
func ImportSnapshot(ctx context.Context, db *sql.DB, dialect string, r io.Reader, targetTenantID string) (*SnapshotManifest, error) {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("%w: gzip decompression error: %v", ErrInvalidSnapshot, err)
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	var manifest *SnapshotManifest
	tables := make(map[string]TableExport)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		data, err := io.ReadAll(io.LimitReader(tarReader, 256<<20))
		if err != nil {
			return nil, err
		}

		if header.Name == "manifest.json" {
			var m SnapshotManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, fmt.Errorf("%w: invalid manifest: %v", ErrInvalidSnapshot, err)
			}
			manifest = &m
		} else if strings.HasPrefix(header.Name, "tables/") && strings.HasSuffix(header.Name, ".json") {
			var table TableExport
			if err := json.Unmarshal(data, &table); err != nil {
				return nil, fmt.Errorf("%w: invalid table file %s: %v", ErrInvalidSnapshot, header.Name, err)
			}
			tables[table.Name] = table
		}
	}

	if manifest == nil {
		return nil, fmt.Errorf("%w: archive missing manifest.json", ErrInvalidSnapshot)
	}

	effectiveTenantID := manifest.TenantID
	if strings.TrimSpace(targetTenantID) != "" {
		effectiveTenantID = strings.TrimSpace(targetTenantID)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if dialect == "postgres" {
		scope, err := NewScope(effectiveTenantID, effectiveTenantID)
		if err == nil {
			_ = BindPostgresTransaction(ctx, tx, scope)
		}
	}

	for tableName, table := range tables {
		if len(table.Rows) == 0 {
			continue
		}

		for _, row := range table.Rows {
			row["tenant_id"] = effectiveTenantID

			cols := make([]string, 0, len(row))
			placeholders := make([]string, 0, len(row))
			args := make([]any, 0, len(row))

			idx := 1
			for col, val := range row {
				cols = append(cols, col)
				if dialect == "postgres" {
					placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
				} else {
					placeholders = append(placeholders, "?")
				}
				args = append(args, val)
				idx++
			}

			var insertStmt string
			if dialect == "postgres" {
				insertStmt = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING", tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
			} else if dialect == "mysql" {
				insertStmt = fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)", tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
			} else {
				insertStmt = fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)", tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
			}

			if _, err := tx.ExecContext(ctx, insertStmt, args...); err != nil {
				if isMissingTableError(err) {
					break
				}
				return nil, fmt.Errorf("insert into %s: %w", tableName, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	manifest.TenantID = effectiveTenantID
	return manifest, nil
}

var _ = bytes.NewReader
