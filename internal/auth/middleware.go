package auth

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

// Middleware extracts the session cookie, validates the session, and attaches the user to the request context.
func Middleware(authSvc *AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			sess, u, err := authSvc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				ClearSessionCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			ctx := WithSession(r.Context(), sess)
			ctx = WithUser(ctx, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Authorizer returns a router.Authorizer that enforces authentication and optional role restrictions.
func Authorizer(roles ...Role) router.Authorizer {
	allowedRoles := make(map[Role]bool)
	for _, r := range roles {
		allowedRoles[r] = true
	}

	return func(r *http.Request) bool {
		u := UserFromContext(r.Context())
		if u == nil || !u.Active {
			return false
		}
		if len(allowedRoles) == 0 {
			return true
		}
		return allowedRoles[u.Role]
	}
}
