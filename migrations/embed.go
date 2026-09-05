package migrations

import "embed"

// FS contains immutable, dialect-specific migration inputs.
//
//go:embed */*.sql */*.yaml
var FS embed.FS
