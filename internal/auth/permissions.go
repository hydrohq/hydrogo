package auth

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
)

const (
	PermissionContentRead   = "admin.content.read"
	PermissionContentWrite  = "admin.content.write"
	PermissionCommerceRead  = "admin.commerce.read"
	PermissionCommerceWrite = "admin.commerce.write"
	PermissionMembersManage = "admin.members.manage"
	PermissionUsersManage   = "admin.users.manage"
	PermissionMediaManage   = "admin.media.manage"
	PermissionSiteManage    = "admin.site.manage"
	PermissionOperations    = "admin.operations.manage"
)

var KnownPermissions = []string{
	PermissionContentRead, PermissionContentWrite, PermissionCommerceRead,
	PermissionCommerceWrite, PermissionMembersManage, PermissionUsersManage,
	PermissionMediaManage, PermissionSiteManage, PermissionOperations,
}

const MaxPermissionCatalogEntries = 512

var permissionNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{1,127}$`)

// PermissionCatalog is an immutable allowlist composed at Edition startup.
// Core permissions are always present; generated resource permissions are
// added explicitly by the Edition composition root.
type PermissionCatalog struct {
	names []string
	set   map[string]struct{}
}

// NewPermissionCatalog combines built-in permissions with explicitly enabled
// Edition permissions. It grants no role access by itself.
func NewPermissionCatalog(editionPermissions ...string) (*PermissionCatalog, error) {
	if len(editionPermissions) > MaxPermissionCatalogEntries {
		return nil, errors.New("permission catalog exceeds entry limit")
	}
	set := make(map[string]struct{}, len(KnownPermissions)+len(editionPermissions))
	for _, permission := range append(append([]string(nil), KnownPermissions...), editionPermissions...) {
		if permission != strings.TrimSpace(permission) || !permissionNamePattern.MatchString(permission) || strings.Contains(permission, "..") {
			return nil, errors.New("invalid permission catalog entry")
		}
		set[permission] = struct{}{}
		if len(set) > MaxPermissionCatalogEntries {
			return nil, errors.New("permission catalog exceeds entry limit")
		}
	}
	names := make([]string, 0, len(set))
	for permission := range set {
		names = append(names, permission)
	}
	sort.Strings(names)
	return &PermissionCatalog{names: names, set: set}, nil
}

// Names returns a detached, deterministic catalog snapshot.
func (c *PermissionCatalog) Names() []string {
	if c == nil {
		return nil
	}
	return append([]string(nil), c.names...)
}

// Contains reports whether a permission was declared by Core or the Edition.
func (c *PermissionCatalog) Contains(permission string) bool {
	if c == nil {
		return false
	}
	_, ok := c.set[permission]
	return ok
}

// Allows evaluates the built-in role defaults and user override only after
// confirming the permission belongs to this Edition's active catalog.
func (c *PermissionCatalog) Allows(user *User, permission string) bool {
	return c.Contains(permission) && Can(user, permission)
}

type PermissionStore interface {
	LoadPermissionOverrides(ctx context.Context, userID string) (map[string]bool, error)
	ReplacePermissionOverrides(ctx context.Context, userID string, overrides map[string]bool) error
}

func roleAllows(role Role, permission string) bool {
	if role == RoleAdmin {
		return true
	}
	switch role {
	case RoleEditor:
		return permission == PermissionContentRead || permission == PermissionContentWrite || permission == PermissionCommerceRead || permission == PermissionMediaManage
	case RoleViewer:
		return permission == PermissionContentRead || permission == PermissionCommerceRead
	default:
		return false
	}
}

func Can(user *User, permission string) bool {
	if user == nil || !user.Active {
		return false
	}
	if user.PermissionOverrides != nil {
		if allowed, ok := user.PermissionOverrides[permission]; ok {
			return allowed
		}
	}
	return roleAllows(user.Role, permission)
}
