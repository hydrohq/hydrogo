package ui

import (
	"context"
	"io"

	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func RenderAdminAudit(w io.Writer, locale string, page designsystem.AdminAuditPageProps) error {
	if err := AdminStartLocale(w, "Audit history", "Security", "audit", locale); err != nil {
		return err
	}
	if err := designsystem.AdminAuditPage(page).Render(context.Background(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}
