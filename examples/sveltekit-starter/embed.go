// Package sveltekitstarter embeds the SvelteKit starter so `hydro new
// --starter sveltekit` can copy it into a new project's frontend directory.
package sveltekitstarter

import "embed"

// FS holds the starter's source files (no node_modules or build output).
//
//go:embed all:src .env.example .gitignore Dockerfile README.md package.json package-lock.json svelte.config.js tsconfig.json vite.config.ts
var FS embed.FS
