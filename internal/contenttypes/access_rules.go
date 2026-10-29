package contenttypes

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Principal identifies the caller evaluated in access rules.
type Principal struct {
	ID        string
	Email     string
	Role      string
	IsAdmin   bool
	IsMember  bool
	Anonymous bool
}

// AccessRules defines PocketBase-style row-level access rules for a content type.
type AccessRules struct {
	List   string `json:"list,omitempty" yaml:"list"`
	Read   string `json:"read,omitempty" yaml:"read"`
	Create string `json:"create,omitempty" yaml:"create"`
	Update string `json:"update,omitempty" yaml:"update"`
	Delete string `json:"delete,omitempty" yaml:"delete"`
}

// normalizeRule replaces PocketBase-style @auth.* and record.* references
// with standard identifiers supported by the expression evaluator.
func normalizeRule(rule string) string {
	r := rule
	r = strings.ReplaceAll(r, "@request.auth.id", "auth_id")
	r = strings.ReplaceAll(r, "@auth.id", "auth_id")
	r = strings.ReplaceAll(r, "@request.auth.email", "auth_email")
	r = strings.ReplaceAll(r, "@auth.email", "auth_email")
	r = strings.ReplaceAll(r, "@request.auth.role", "auth_role")
	r = strings.ReplaceAll(r, "@auth.role", "auth_role")
	r = strings.ReplaceAll(r, "@request.auth.is_admin", "auth_is_admin")
	r = strings.ReplaceAll(r, "@auth.is_admin", "auth_is_admin")
	r = strings.ReplaceAll(r, "@auth.isAdmin", "auth_is_admin")
	r = strings.ReplaceAll(r, "@request.auth.is_member", "auth_is_member")
	r = strings.ReplaceAll(r, "@auth.is_member", "auth_is_member")
	r = strings.ReplaceAll(r, "@auth.isMember", "auth_is_member")
	r = strings.ReplaceAll(r, "record.", "")
	return r
}

var authFields = map[string]bool{
	"auth_id":        true,
	"auth_email":     true,
	"auth_role":      true,
	"auth_is_admin":  true,
	"auth_is_member": true,
}

var systemRecordFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"version":    true,
	"status":     true,
	"locale":     true,
}

func (d Definition) validateAccessRules(schema collections.Schema) error {
	if d.AccessRules == nil {
		return nil
	}
	rules := []struct {
		op   string
		rule string
	}{
		{"list", d.AccessRules.List},
		{"read", d.AccessRules.Read},
		{"create", d.AccessRules.Create},
		{"update", d.AccessRules.Update},
		{"delete", d.AccessRules.Delete},
	}
	for _, item := range rules {
		if err := ValidateAccessRule(item.rule, schema); err != nil {
			return fmt.Errorf("%w: invalid %s access rule: %v", ErrInvalidDefinition, item.op, err)
		}
	}
	return nil
}

// ValidateAccessRule validates that an access rule is syntactically sound
// and only references known auth properties or schema fields.
func ValidateAccessRule(rule string, schema collections.Schema) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil
	}
	switch strings.ToLower(rule) {
	case "public", "true", "false", "admin", "member":
		return nil
	}
	normalized := normalizeRule(rule)
	expr, err := collections.ParseExpr(normalized)
	if err != nil {
		return err
	}
	validFields := make(map[string]bool, len(schema.Fields)+len(systemRecordFields)+len(authFields))
	for k := range authFields {
		validFields[k] = true
	}
	for k := range systemRecordFields {
		validFields[k] = true
	}
	for _, field := range schema.Fields {
		validFields[field.ID] = true
	}
	for _, name := range expr.Fields() {
		if !validFields[name] {
			return fmt.Errorf("unknown field or property %q", name)
		}
	}
	return nil
}

// EvaluateRule evaluates a rule against the caller principal and optional record data.
// For read/list queries, an empty rule defaults to allowed (true) when public.
// For create/update/delete mutations, an empty rule defaults to denied (false) unless admin.
func EvaluateRule(rule string, principal Principal, record map[string]any, isReadOrList bool) (bool, error) {
	if principal.IsAdmin {
		return true, nil
	}
	rule = strings.TrimSpace(rule)
	if rule == "" {
		if isReadOrList {
			return true, nil
		}
		return false, nil
	}
	switch strings.ToLower(rule) {
	case "public", "true":
		return true, nil
	case "false":
		return false, nil
	case "admin":
		return principal.IsAdmin, nil
	case "member":
		return !principal.Anonymous && (principal.IsMember || principal.IsAdmin), nil
	}

	normalized := normalizeRule(rule)
	expr, err := collections.ParseExpr(normalized)
	if err != nil {
		return false, err
	}

	return expr.EvalBool(func(name string) collections.Value {
		switch name {
		case "auth_id":
			return principal.ID
		case "auth_email":
			return principal.Email
		case "auth_role":
			return principal.Role
		case "auth_is_admin":
			return principal.IsAdmin
		case "auth_is_member":
			return principal.IsMember
		}
		if record == nil {
			return nil
		}
		val, ok := record[name]
		if !ok {
			return nil
		}
		return toExprValue(val)
	})
}

func toExprValue(val any) collections.Value {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		return v
	case *big.Rat:
		return v
	case int:
		return big.NewRat(int64(v), 1)
	case int64:
		return big.NewRat(v, 1)
	case int32:
		return big.NewRat(int64(v), 1)
	case float64:
		return new(big.Rat).SetFloat64(v)
	case float32:
		return new(big.Rat).SetFloat64(float64(v))
	case time.Time:
		return v.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}
