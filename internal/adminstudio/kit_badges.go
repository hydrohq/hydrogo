package adminstudio

import (
	"context"
	"io"
	"sort"
)

// RenderKitBadges renders a compact, escaped capability summary for admin
// collection headers. It intentionally exposes names only, never provider or
// credential details.
func RenderKitBadges(w io.Writer, kits []Kit) error {
	ordered := append([]Kit(nil), kits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return kitBadgesView(ordered).Render(context.Background(), w)
}
