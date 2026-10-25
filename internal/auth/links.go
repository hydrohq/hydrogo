package auth

import (
	"net/url"
	"strings"
)

// The admin panel hosts the two pages a person lands on from an e-mail: the
// form that sets a new password and the one that accepts an invitation. Both
// are public routes of the admin SPA (see admin/src/routes/account) that post
// to the JSON endpoints below. Projects with their own front end can build
// the same links with a different path.
const (
	PasswordResetPagePath = "/admin/reset-password"
	InviteAcceptPagePath  = "/admin/accept-invite"
	LoginPagePath         = "/admin/login"
)

func pageLink(origin, path, token string) string {
	return strings.TrimRight(strings.TrimSpace(origin), "/") + path + "?token=" + url.QueryEscape(token)
}

// PasswordResetLink is the page where a reset token is redeemed.
func PasswordResetLink(origin, token string) string {
	return pageLink(origin, PasswordResetPagePath, token)
}

// InviteAcceptLink is the page where an invitation token is redeemed.
func InviteAcceptLink(origin, token string) string {
	return pageLink(origin, InviteAcceptPagePath, token)
}
