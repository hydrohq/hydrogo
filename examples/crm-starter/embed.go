// Package crmstarter embeds the CRM panel starter so `hydro new
// --starter crm` can copy it into a new project's frontend directory.
package crmstarter

import "embed"

// FS holds the starter's source files (no node_modules or build output).
//
//go:embed all:src scripts .env.example .gitignore Dockerfile README.md package.json package-lock.json svelte.config.js tsconfig.json vite.config.ts
var FS embed.FS
