package members

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

// OAuthResolver maps a verified social identity to a member account. Unlike
// the admin resolver (internal/auth/sqlstore.OAuthIdentityStore), which only
// ever links to an existing active admin account, members are self-service:
// a provider identity with no existing match (by identity, then by email)
// provisions a new active member. That auto-provisioning policy lives in
// the sqlstore implementation, not here.
type OAuthResolver interface {
	ResolveOAuth(context.Context, auth.OAuthIdentity) (*Member, error)
}
