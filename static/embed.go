package static

import "embed"

// FS contains release-owned browser libraries (vendor/) and the admin's own
// static assets (admin/). Uploaded media and themes stay external runtime
// resources, but editor assets travel with the binary.
//
//go:embed vendor admin
var FS embed.FS
