package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrRoleNotFound = errors.New("role not found")
	ErrRoleInvalid  = errors.New("invalid role definition")
)

var roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// ValidCustomRoleName validates identifier format for custom roles.
func ValidCustomRoleName(name string) bool {
	return roleNamePattern.MatchString(name)
}

// CustomRole represents a tenant-defined access role with granular permissions.
type CustomRole struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"tenant_id,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Permissions []RolePermission `json:"permissions"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// RolePermission defines a capability over a specific resource and action.
type RolePermission struct {
	Resource string `json:"resource"` // e.g. "collection:products", "media", "users", "settings", "*"
	Action   string `json:"action"`   // e.g. "read", "create", "update", "delete", "publish", "*"
}

// RoleStore defines database persistence for custom roles and their permission bindings.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]CustomRole, error)
	GetRole(ctx context.Context, id string) (CustomRole, error)
	SaveRole(ctx context.Context, role CustomRole) error
	DeleteRole(ctx context.Context, id string) error
}

// CanResource evaluates whether the user has permission to perform action on resource.
func CanResource(user *User, resource, action string) bool {
	if user == nil || !user.Active {
		return false
	}
	if user.Role == RoleAdmin {
		return true
	}

	action = strings.ToLower(strings.TrimSpace(action))
	resource = strings.ToLower(strings.TrimSpace(resource))

	// Built-in standard roles
	switch user.Role {
	case RoleEditor:
		if action == "read" {
			return true
		}
		if action == "create" || action == "update" || action == "delete" || action == "publish" {
			return resource != "users" && resource != "settings"
		}
	case RoleViewer:
		return action == "read"
	}

	// Dynamic / custom role permissions on the user object if loaded
	if user.Permissions != nil {
		for _, perm := range user.Permissions {
			r := strings.ToLower(strings.TrimSpace(perm.Resource))
			a := strings.ToLower(strings.TrimSpace(perm.Action))
			if (r == "*" || r == resource) && (a == "*" || a == action) {
				return true
			}
		}
	}

	return false
}
