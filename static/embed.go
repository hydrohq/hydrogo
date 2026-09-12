package static

import "embed"

// FS contains release-owned browser libraries. Uploaded media and themes stay
// external runtime resources, but editor assets travel with the binary.
//
//go:embed vendor
var FS embed.FS
