package ui

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui/shell"
)

// AdminScriptHashes lists the CSP hashes of the inline scripts the admin may
// render. The frame's own behaviour is loaded from /admin-assets/js/ and needs
// no hash; only its pre-paint theme bootstrap is inline.
func AdminScriptHashes() ([]string, error) {
	var hashes []string
	for _, render := range []func(*bytes.Buffer) error{
		func(out *bytes.Buffer) error { return shell.Head(shell.View{}).Render(context.Background(), out) },
		func(out *bytes.Buffer) error { return adminControlTemplate.Execute(out, AdminControlData{}) },
		func(out *bytes.Buffer) error { return adminReportsPathTemplate.Execute(out, AdminDashboardData{}) },
		func(out *bytes.Buffer) error {
			return RenderAdminArea(out, AdminAreaData{Area: "developer", Locale: "tr"})
		},
	} {
		var document bytes.Buffer
		if err := render(&document); err != nil {
			return nil, fmt.Errorf("render Admin script catalog: %w", err)
		}
		scripts, _ := security.InlineCSPHashes(document.Bytes())
		hashes = append(hashes, scripts...)
	}
	return uniqueScriptHashes(hashes), nil
}

func uniqueScriptHashes(hashes []string) []string {
	seen := make(map[string]struct{}, len(hashes))
	unique := make([]string, 0, len(hashes))
	for _, source := range hashes {
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		unique = append(unique, source)
	}
	return unique
}
