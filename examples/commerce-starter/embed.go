// Package commercestarter embeds the commerce storefront starter so `hydro
// new --starter commerce` can copy it into a new project's frontend
// directory.
package commercestarter

import "embed"

// FS holds the starter's source files (no node_modules or build output).
//
//go:embed all:src .env.example .gitignore Dockerfile README.md package.json package-lock.json svelte.config.js tsconfig.json vite.config.ts
var FS embed.FS
