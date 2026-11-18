package storefront

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedDistFS embed.FS

// EmbeddedDist returns the embedded storefront filesystem sub-rooted at "dist".
func EmbeddedDist() (fs.FS, error) {
	return fs.Sub(embeddedDistFS, "dist")
}
