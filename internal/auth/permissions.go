package auth

import "context"

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
