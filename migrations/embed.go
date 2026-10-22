package migrations

import (
	"embed"
	"io/fs"
)

// files contains immutable, dialect-specific migration inputs: one directory
// per migration, which is also the migration's ID.
//
//go:embed */*.sql */*.yaml
var files embed.FS

// baselineFiles holds the generated schema baseline (ADR 0009). It lives next
// to, not among, the migration directories so listing FS still yields exactly
// the migration IDs.
//
//go:embed baseline
var baselineFiles embed.FS

// Bundle is the migration chain plus the baseline that can stand in for it on
// databases that have not applied the covered migrations yet.
type Bundle struct {
	embed.FS
}

// FS is the embedded migration bundle. It reads like an fs.FS over the
// migration directories; the database package also asks it for BaselineFS.
var FS = Bundle{FS: files}

// BaselineFS returns the baseline manifest and its per-group, per-dialect SQL.
func (Bundle) BaselineFS() fs.FS {
	sub, err := fs.Sub(baselineFiles, "baseline")
	if err != nil {
		panic(err)
	}
	return sub
}
