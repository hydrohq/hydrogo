// Package newsstarter embeds the Turkish news portal starter (HydroPost)
// so `hydro new --starter news` can copy it into a new project's frontend directory.
package newsstarter

import "embed"

// FS holds the starter's source files (no node_modules or build output).
//
//go:embed all:src all:static scripts .env.example .gitignore .npmrc AGENTS.md README.md package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.json vite.config.ts
var FS embed.FS
