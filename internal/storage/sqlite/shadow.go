package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrShadowInvalid     = errors.New("invalid SQLite shadow migration")
	ErrShadowUnsupported = errors.New("SQLite shadow migration is not supported for this table")
)

const shadowStateTable = "hydro_sqlite_shadow_migrations"

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var migrationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_]*$`)

// ShadowColumn maps one old-table projection to its replacement column.
// Expressions are trusted, versioned migration source, not user input.
type ShadowColumn struct {
	Name    string `yaml:"name" json:"name"`
	Copy    string `yaml:"copy" json:"copy"`
	OnWrite string `yaml:"on_write" json:"on_write"`
}

// ShadowMigration describes a resumable, batched SQLite table rebuild. The
// replacement CREATE statement must contain exactly one {{table}} token.
type ShadowMigration struct {
	Table     string         `yaml:"table" json:"table"`
	CreateSQL string         `yaml:"create_sql" json:"create_sql"`
	Columns   []ShadowColumn `yaml:"columns" json:"columns"`
	BatchSize int            `yaml:"batch_size" json:"batch_size"`
	AfterSwap []string       `yaml:"after_swap" json:"after_swap"`
}

// RebuildTable copies bounded batches while source-table triggers mirror
// writes, then swaps tables and records migration completion in one cutover
// transaction. record must update the caller's migration ledger using conn;
// that update is committed atomically with the table swap.
func RebuildTable(ctx context.Context, conn *sql.Conn, migrationID string, spec ShadowMigration, record func(context.Context, *sql.Conn) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if conn == nil || record == nil || !migrationIDPattern.MatchString(migrationID) || !identifierPattern.MatchString(spec.Table) || strings.Count(spec.CreateSQL, "{{table}}") != 1 || strings.Contains(spec.CreateSQL, ";") || len(spec.Columns) == 0 {
		return ErrShadowInvalid
	}
	if spec.BatchSize <= 0 || spec.BatchSize > 1000 {
		spec.BatchSize = 200
	}
	var journalMode string
	if err := conn.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		return err
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("%w: WAL journal mode is required", ErrShadowUnsupported)
	}
	var foreignKeys int
	if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		return err
	}
	if foreignKeys != 0 {
		if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
			return err
		}
		defer func() {
			restoreCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, _ = conn.ExecContext(restoreCtx, `PRAGMA foreign_keys=ON`)
		}()
	}
	seen := map[string]bool{}
	for _, column := range spec.Columns {
		if !identifierPattern.MatchString(column.Name) || column.Copy == "" || column.OnWrite == "" || seen[strings.ToLower(column.Name)] {
			return ErrShadowInvalid
		}
		seen[strings.ToLower(column.Name)] = true
	}
	for _, statement := range spec.AfterSwap {
		if strings.TrimSpace(statement) == "" {
			return ErrShadowInvalid
		}
	}
	shadowTable := spec.Table + "__hydro_shadow_" + migrationID
	if !identifierPattern.MatchString(shadowTable) {
		return ErrShadowInvalid
	}
	if err := prepare(ctx, conn, migrationID, spec, shadowTable); err != nil {
		return err
	}
	if err := copyBatches(ctx, conn, migrationID, spec, shadowTable); err != nil {
		return err
	}
	return cutover(ctx, conn, migrationID, spec, shadowTable, record)
}

func prepare(ctx context.Context, conn *sql.Conn, migrationID string, spec ShadowMigration, shadowTable string) error {
	descriptorChecksum, err := Checksum(spec)
	if err != nil {
		return err
	}
	return immediate(ctx, conn, func() error {
		if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+quote(shadowStateTable)+` (migration_id TEXT PRIMARY KEY, source_table TEXT NOT NULL, shadow_table TEXT NOT NULL, descriptor_checksum TEXT NOT NULL, cursor_value INTEGER, has_cursor INTEGER NOT NULL DEFAULT 0, phase TEXT NOT NULL)`); err != nil {
			return err
		}
		var existingSource, existingShadow, existingChecksum, phase string
		err := conn.QueryRowContext(ctx, `SELECT source_table, shadow_table, descriptor_checksum, phase FROM `+quote(shadowStateTable)+` WHERE migration_id=?`, migrationID).Scan(&existingSource, &existingShadow, &existingChecksum, &phase)
		if err == nil {
			if existingSource != spec.Table || existingShadow != shadowTable || existingChecksum != descriptorChecksum || (phase != "copy" && phase != "cutover") {
				return ErrShadowInvalid
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := requireOrdinaryRowIDTable(ctx, conn, spec.Table); err != nil {
			return err
		}
		createSQL := strings.Replace(spec.CreateSQL, "{{table}}", quote(shadowTable), 1)
		if _, err := conn.ExecContext(ctx, createSQL); err != nil {
			return err
		}
		if err := requireOrdinaryRowIDTable(ctx, conn, shadowTable); err != nil {
			return err
		}
		if err := rejectIntegerPrimaryKey(ctx, conn, shadowTable); err != nil {
			return err
		}
		sourceRowID, err := rowIDAlias(ctx, conn, spec.Table)
		if err != nil {
			return err
		}
		shadowRowID, err := rowIDAlias(ctx, conn, shadowTable)
		if err != nil {
			return err
		}
		if err := createMirrorTriggers(ctx, conn, migrationID, spec, shadowTable, sourceRowID, shadowRowID); err != nil {
			return err
		}
		_, err = conn.ExecContext(ctx, `INSERT INTO `+quote(shadowStateTable)+` (migration_id, source_table, shadow_table, descriptor_checksum, phase) VALUES (?, ?, ?, ?, 'copy')`, migrationID, spec.Table, shadowTable, descriptorChecksum)
		return err
	})
}

func copyBatches(ctx context.Context, conn *sql.Conn, migrationID string, spec ShadowMigration, shadowTable string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done := false
		err := immediate(ctx, conn, func() error {
			var cursor sql.NullInt64
			var hasCursor int
			var phase string
			if err := conn.QueryRowContext(ctx, `SELECT cursor_value, has_cursor, phase FROM `+quote(shadowStateTable)+` WHERE migration_id=?`, migrationID).Scan(&cursor, &hasCursor, &phase); err != nil {
				return err
			}
			if phase == "cutover" {
				done = true
				return nil
			}
			rowID, err := rowIDAlias(ctx, conn, spec.Table)
			if err != nil {
				return err
			}
			shadowRowID, err := rowIDAlias(ctx, conn, shadowTable)
			if err != nil {
				return err
			}
			query := `SELECT ` + quote(rowID) + ` FROM ` + quote(spec.Table)
			args := []any{}
			if hasCursor != 0 {
				query += ` WHERE ` + quote(rowID) + `>?`
				args = append(args, cursor.Int64)
			}
			query += ` ORDER BY ` + quote(rowID) + ` LIMIT ?`
			args = append(args, spec.BatchSize)
			rows, err := conn.QueryContext(ctx, query, args...)
			if err != nil {
				return err
			}
			var batch []int64
			for rows.Next() {
				var value int64
				if err := rows.Scan(&value); err != nil {
					rows.Close()
					return err
				}
				batch = append(batch, value)
			}
			rowsErr := rows.Err()
			_ = rows.Close()
			if rowsErr != nil {
				return rowsErr
			}
			if len(batch) == 0 {
				_, err := conn.ExecContext(ctx, `UPDATE `+quote(shadowStateTable)+` SET phase='cutover' WHERE migration_id=?`, migrationID)
				done = err == nil
				return err
			}
			if err := copyBatch(ctx, conn, spec, shadowTable, rowID, shadowRowID, batch[0], batch[len(batch)-1]); err != nil {
				return err
			}
			_, err = conn.ExecContext(ctx, `UPDATE `+quote(shadowStateTable)+` SET cursor_value=?, has_cursor=1 WHERE migration_id=?`, batch[len(batch)-1], migrationID)
			return err
		})
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func copyBatch(ctx context.Context, conn *sql.Conn, spec ShadowMigration, shadowTable, sourceRowID, shadowRowID string, low, high int64) error {
	columns := make([]string, 0, len(spec.Columns)+1)
	expressions := make([]string, 0, len(spec.Columns)+1)
	columns = append(columns, quote(shadowRowID))
	expressions = append(expressions, quote(sourceRowID))
	for _, column := range spec.Columns {
		columns = append(columns, quote(column.Name))
		expressions = append(expressions, "("+column.Copy+")")
	}
	assignments := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		assignments = append(assignments, quote(column.Name)+"=excluded."+quote(column.Name))
	}
	q := `INSERT INTO ` + quote(shadowTable) + ` (` + strings.Join(columns, ",") + `) SELECT ` + strings.Join(expressions, ",") + ` FROM ` + quote(spec.Table) + ` WHERE ` + quote(sourceRowID) + ` BETWEEN ? AND ? ORDER BY ` + quote(sourceRowID) + ` ON CONFLICT(` + quote(shadowRowID) + `) DO UPDATE SET ` + strings.Join(assignments, ",")
	_, err := conn.ExecContext(ctx, q, low, high)
	return err
}

func cutover(ctx context.Context, conn *sql.Conn, migrationID string, spec ShadowMigration, shadowTable string, record func(context.Context, *sql.Conn) error) error {
	return immediate(ctx, conn, func() error {
		var phase string
		if err := conn.QueryRowContext(ctx, `SELECT phase FROM `+quote(shadowStateTable)+` WHERE migration_id=?`, migrationID).Scan(&phase); err != nil {
			return err
		}
		if phase != "cutover" {
			return ErrShadowInvalid
		}
		for _, suffix := range []string{"ins", "upd", "del"} {
			if _, err := conn.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+quote(triggerName(migrationID, suffix))); err != nil {
				return err
			}
		}
		if _, err := conn.ExecContext(ctx, `DROP TABLE `+quote(spec.Table)); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, `ALTER TABLE `+quote(shadowTable)+` RENAME TO `+quote(spec.Table)); err != nil {
			return err
		}
		for _, statement := range spec.AfterSwap {
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		rows, err := conn.QueryContext(ctx, `PRAGMA foreign_key_check`)
		if err != nil {
			return err
		}
		if rows.Next() {
			_ = rows.Close()
			return errors.New("SQLite shadow migration failed foreign-key validation")
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		_ = rows.Close()
		if err := record(ctx, conn); err != nil {
			return err
		}
		_, err = conn.ExecContext(ctx, `DELETE FROM `+quote(shadowStateTable)+` WHERE migration_id=?`, migrationID)
		return err
	})
}

func createMirrorTriggers(ctx context.Context, conn *sql.Conn, migrationID string, spec ShadowMigration, shadowTable, sourceRowID, shadowRowID string) error {
	targets := make([]string, 0, len(spec.Columns)+1)
	values := make([]string, 0, len(spec.Columns)+1)
	for _, column := range spec.Columns {
		targets = append(targets, quote(column.Name))
		values = append(values, "("+column.OnWrite+")")
	}
	write := `INSERT OR REPLACE INTO ` + quote(shadowTable) + ` (` + quote(shadowRowID) + `,` + strings.Join(targets, ",") + `) VALUES (NEW.` + quote(sourceRowID) + `,` + strings.Join(values, ",") + `);`
	triggers := []string{
		`CREATE TRIGGER ` + quote(triggerName(migrationID, "ins")) + ` AFTER INSERT ON ` + quote(spec.Table) + ` BEGIN ` + write + ` END`,
		`CREATE TRIGGER ` + quote(triggerName(migrationID, "upd")) + ` AFTER UPDATE ON ` + quote(spec.Table) + ` BEGIN DELETE FROM ` + quote(shadowTable) + ` WHERE ` + quote(shadowRowID) + `=OLD.` + quote(sourceRowID) + `; ` + write + ` END`,
		`CREATE TRIGGER ` + quote(triggerName(migrationID, "del")) + ` AFTER DELETE ON ` + quote(spec.Table) + ` BEGIN DELETE FROM ` + quote(shadowTable) + ` WHERE ` + quote(shadowRowID) + `=OLD.` + quote(sourceRowID) + `; END`,
	}
	for _, trigger := range triggers {
		if _, err := conn.ExecContext(ctx, trigger); err != nil {
			return err
		}
	}
	return nil
}

func immediate(ctx context.Context, conn *sql.Conn, fn func() error) error {
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, _ = conn.ExecContext(rollbackCtx, `ROLLBACK`)
		}
	}()
	if err := fn(); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	committed = true
	return nil
}

func requireOrdinaryRowIDTable(ctx context.Context, conn *sql.Conn, table string) error {
	rows, err := conn.QueryContext(ctx, `PRAGMA table_list(`+quote(table)+`)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, name, kind string
		var columns, withoutRowID, strict int
		if err := rows.Scan(&schema, &name, &kind, &columns, &withoutRowID, &strict); err != nil {
			return err
		}
		if name == table && kind == "table" && withoutRowID == 0 {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return ErrShadowUnsupported
}

func rejectIntegerPrimaryKey(ctx context.Context, conn *sql.Conn, table string) error {
	rows, err := conn.QueryContext(ctx, `PRAGMA table_info(`+quote(table)+`)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if primaryKey > 0 && strings.EqualFold(strings.TrimSpace(kind), "INTEGER") {
			return ErrShadowUnsupported
		}
	}
	return rows.Err()
}

func rowIDAlias(ctx context.Context, conn *sql.Conn, table string) (string, error) {
	rows, err := conn.QueryContext(ctx, `PRAGMA table_info(`+quote(table)+`)`)
	if err != nil {
		return "", err
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return "", err
		}
		columns[strings.ToLower(name)] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return "", err
	}
	_ = rows.Close()
	for _, alias := range []string{"rowid", "_rowid_", "oid"} {
		if !columns[alias] {
			return alias, nil
		}
	}
	return "", ErrShadowUnsupported
}

func triggerName(migrationID, suffix string) string {
	return "hydro_shadow_" + migrationID + "_" + suffix
}
func quote(identifier string) string { return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"` }

// Checksum returns a stable digest of the shadow migration descriptor, used
// by migrationguard and the runtime migration ledger.
func Checksum(spec ShadowMigration) (string, error) {
	if spec.BatchSize <= 0 || spec.BatchSize > 1000 {
		spec.BatchSize = 200
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
