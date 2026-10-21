// Package contenttypedata embeds the stock content type definitions so the hydro
// binary can scaffold new projects without the source tree.
package contenttypedata

import "embed"

//go:embed *.yaml crm/*.yaml erp/*.yaml
var FS embed.FS
