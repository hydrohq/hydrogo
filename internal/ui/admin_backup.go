package ui

import (
	"context"
	"io"

	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func RenderAdminBackup(w io.Writer, csrf, locale string, recent []designsystem.AdminBackupItem, cloudEnabled bool, publicKey string) error {
	if err := AdminStartLocale(w, "Backup", "Operations", "backup", locale); err != nil {
		return err
	}
	if err := designsystem.AdminBackupPage(designsystem.AdminBackupPageProps{CSRF: csrf, Locale: locale, Recent: recent, CloudEnabled: cloudEnabled, PublicKey: publicKey}).Render(context.Background(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}
