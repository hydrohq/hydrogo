package auth

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

// Middleware authenticates the request and attaches the user and principal
// to the context. A HydroGo token header (Authorization: Bearer hyd_… or
// X-API-Key) is authoritative: the session cookie is ignored whether or not
// the token validates, so a stolen cookie can never widen a key's scope.
// Read-only keys are refused on mutating methods here, before routing.
func Middleware(authSvc *AuthService, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if credential, isToken := APITokenFromRequest(r); isToken {
				user, token, err := authSvc.ValidateAPIToken(r.Context(), credential)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				principal := Principal{Kind: PrincipalToken, Scopes: token.Scopes, TokenID: token.ID}
				if token.Kind == APITokenKey {
					principal.Kind = PrincipalAPIKey
				}
				if !principal.Allows(r.Method) {
					api.WriteError(w, r, http.StatusForbidden, "this credential is read-only", nil)
					return
				}
				if principal.Kind == PrincipalAPIKey {
					user = apiKeyUser(user)
				}
				ctx := WithUser(r.Context(), user)
				ctx = WithPrincipal(ctx, principal)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			sess, u, err := authSvc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				ClearSessionCookie(w, secure)
				next.ServeHTTP(w, r)
				return
			}

			ctx := WithSession(r.Context(), sess)
			ctx = WithUser(ctx, u)
			ctx = WithPrincipal(ctx, Principal{Kind: PrincipalSession})
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

// apiKeyUser bounds an API key to viewer powers whatever its owner's role.
// Keys live in frontend servers and build pipelines; a key an admin created
// must not read the member list, the audit log or user accounts just
// because its owner may. Overrides only ever narrow: a permission the owner
// was explicitly denied stays denied.
func apiKeyUser(owner *User) *User {
	if owner == nil {
		return nil
	}
	bounded := *owner
	bounded.Role = RoleViewer
	overrides := make(map[string]bool, len(KnownPermissions))
	for _, permission := range KnownPermissions {
		overrides[permission] = false
	}
	for _, permission := range []string{PermissionContentRead, PermissionCommerceRead} {
		overrides[permission] = Can(owner, permission)
	}
	bounded.PermissionOverrides = overrides
	return &bounded
}
