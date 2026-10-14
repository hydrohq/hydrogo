package main

import (
	"bytes"
	"context"
	"net/url"
)

// loginSuccessMessage is the confirmation shown on /login after an accepted
// invitation or a password reset, or "" for a plain sign-in.
func loginSuccessMessage(query url.Values, locale string) string {
	switch {
	case query.Get("invite_accepted") == "1":
		if locale == "tr" {
			return "Hesabın hazır. Az önce belirlediğin parolayla giriş yap."
		}
		return "Your account is ready. Sign in with the password you just created."
	case query.Get("password_reset") == "1":
		if locale == "tr" {
			return "Parolan güncellendi. Yeni parolanla giriş yap."
		}
		return "Your password has been updated. Sign in with your new password."
	}
	return ""
}

// withLoginSuccess fills the login page's empty status region with message.
// The page is left unchanged when message is empty or the region is missing.
func withLoginSuccess(page []byte, message string) []byte {
	if message == "" {
		return page
	}
	var placeholder, status bytes.Buffer
	if loginStatusPlaceholder().Render(context.Background(), &placeholder) != nil || loginStatusSuccess(message).Render(context.Background(), &status) != nil {
		return page
	}
	return bytes.Replace(page, placeholder.Bytes(), status.Bytes(), 1)
}
