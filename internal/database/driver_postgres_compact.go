//go:build hydro_compact

package database

import (
	"context"
	"database/sql"
	"errors"
)

func openPostgres(_ context.Context, _ string, _ OpenOptions) (*sql.DB, error) {
	return nil, errors.New(`driver "pgx" is not included in this build (use tags or non-compact build)`)
}
