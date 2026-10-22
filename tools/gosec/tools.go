//go:build tools

// Package gosec pins the security scanner CI runs. It is its own module so the
// scanner's dependencies stay out of the HydroGo module, and so golang.org/x/tools
// can be newer than the one gosec ships with: gosec 2.29.0's own x/tools cannot
// read the export data of Go 1.27 and reports a type error for every package.
//
// Build it with: go build -C tools/gosec -o ../../.tmp/gosec github.com/securego/gosec/v2/cmd/gosec
package gosec

import _ "github.com/securego/gosec/v2/cmd/gosec"
