// Package boutiquestarter embeds the boutique storefront starter so `hydro new
// --starter boutique` can copy it into a new project's frontend directory.
package boutiquestarter

import "embed"

// FS holds the starter's source files (no node_modules or build output).
//
//go:embed all:src index.html README.md package.json package-lock.json tsconfig.json vite.config.ts
var FS embed.FS
