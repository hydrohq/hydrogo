// Package tailwind embeds the statically compiled, build-time Tailwind CSS
// asset described in ADR-0338. This package is intentionally isolated from
// internal/ui: it must never be imported by the admin shell or any of the
// four existing visual families (hydrogo, shadcn, workbench, nova), and the
// compiled CSS must never be loaded on the same page as AdminCSS, since
// Tailwind v4 defines common-named custom properties (e.g. --radius-lg,
// --shadow-md) that can collide with the ADR-0053 semantic token contract.
//
// Regenerate dist/tailwind.css with scripts/build-tailwind.ps1 whenever a
// file under example/ changes; the compiled output is committed so
// `go build ./...` never requires the standalone Tailwind CLI to be present.
package tailwind

import "embed"

//go:embed dist/tailwind.css
var FS embed.FS

// CSS returns the compiled, build-time Tailwind stylesheet for custom
// edition UI authored under custom/ui/tailwind/example. It is not referenced
// by any existing route or admin surface.
func CSS() (string, error) {
	b, err := FS.ReadFile("dist/tailwind.css")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
