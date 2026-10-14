package ui

import (
	"context"
	"io"
)

type AdminControlData struct {
	Email  string
	Nonce  string
	CSRF   string
	Active string
	Locale string
}

func RenderAdminControl(w io.Writer, data AdminControlData) error {
	if err := AdminStartLocale(w, "Yönetim araçları", "Platform yönetimi", "developer", data.Locale); err != nil {
		return err
	}
	if err := adminControlBody(data).Render(context.Background(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}
