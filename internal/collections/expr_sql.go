package collections

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// SQLParamFunc binds a parameter value and returns its dialect placeholder ($N or ?).
type SQLParamFunc func(value any) string

// SQLFieldFunc returns the dialect-specific SQL expression to extract a JSON field.
type SQLFieldFunc func(field string) string

// NormalizeRule replaces PocketBase-style @auth.* and record.* references with identifiers.
func NormalizeRule(rule string) string {
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

// CompileAccessRuleToSQL compiles a declarative row-level access rule into a SQL predicate.
func CompileAccessRuleToSQL(dialect database.Dialect, rawRule string, auth AuthContext, param SQLParamFunc, textExtractor SQLFieldFunc) (string, error) {
	rule := strings.TrimSpace(rawRule)
	if rule == "" || strings.EqualFold(rule, "public") || strings.EqualFold(rule, "true") {
		return "1=1", nil
	}
	if strings.EqualFold(rule, "false") {
		return "1=0", nil
	}
	if strings.EqualFold(rule, "admin") {
		if auth.IsAdmin {
			return "1=1", nil
		}
		return "1=0", nil
	}
	if strings.EqualFold(rule, "member") {
		if auth.IsMember {
			return "1=1", nil
		}
		return "1=0", nil
	}

	normalized := NormalizeRule(rule)
	expr, err := ParseExpr(normalized)
	if err != nil {
		return "", err
	}
	return expr.CompileToSQL(dialect, auth, param, textExtractor)
}

// CompileToSQL translates the parsed AST into a SQL predicate string using the provided callbacks.
func (e *Expr) CompileToSQL(dialect database.Dialect, auth AuthContext, param SQLParamFunc, textExtractor SQLFieldFunc) (string, error) {
	if e == nil || e.root == nil {
		return "1=1", nil
	}
	return compileNodeToSQL(e.root, dialect, auth, param, textExtractor)
}

func compileNodeToSQL(n exprNode, dialect database.Dialect, auth AuthContext, param SQLParamFunc, textExtractor SQLFieldFunc) (string, error) {
	switch node := n.(type) {
	case literalNode:
		switch v := node.value.(type) {
		case string:
			return param(v), nil
		case *big.Rat:
			if v == nil {
				return "NULL", nil
			}
			f, _ := v.Float64()
			return param(f), nil
		case bool:
			switch dialect {
			case database.Postgres:
				if v {
					return "TRUE", nil
				}
				return "FALSE", nil
			default:
				if v {
					return "1", nil
				}
				return "0", nil
			}
		case nil:
			return "NULL", nil
		default:
			return param(v), nil
		}

	case fieldNode:
		switch node.name {
		case "auth_id":
			return param(auth.ID), nil
		case "auth_email":
			return param(auth.Email), nil
		case "auth_role":
			return param(auth.Role), nil
		case "auth_is_admin":
			if auth.IsAdmin {
				return "(1=1)", nil
			}
			return "(1=0)", nil
		case "auth_is_member":
			if auth.IsMember {
				return "(1=1)", nil
			}
			return "(1=0)", nil
		case "id", "created_at", "updated_at", "version":
			return node.name, nil
		default:
			return "COALESCE(" + textExtractor(node.name) + ", '')", nil
		}

	case binaryNode:
		left, err := compileNodeToSQL(node.left, dialect, auth, param, textExtractor)
		if err != nil {
			return "", err
		}
		right, err := compileNodeToSQL(node.right, dialect, auth, param, textExtractor)
		if err != nil {
			return "", err
		}

		switch node.op {
		case "&&":
			return "(" + left + " AND " + right + ")", nil
		case "||":
			return "(" + left + " OR " + right + ")", nil
		case "==":
			return "(" + left + " = " + right + ")", nil
		case "!=":
			return "(" + left + " <> " + right + ")", nil
		case "<":
			return "(" + left + " < " + right + ")", nil
		case "<=":
			return "(" + left + " <= " + right + ")", nil
		case ">":
			return "(" + left + " > " + right + ")", nil
		case ">=":
			return "(" + left + " >= " + right + ")", nil
		default:
			return "", fmt.Errorf("%w: unsupported operator %q in access rule", ErrInvalidExpression, node.op)
		}

	case notNode:
		inner, err := compileNodeToSQL(node.operand, dialect, auth, param, textExtractor)
		if err != nil {
			return "", err
		}
		return "(NOT (" + inner + "))", nil

	case negNode:
		inner, err := compileNodeToSQL(node.operand, dialect, auth, param, textExtractor)
		if err != nil {
			return "", err
		}
		return "(-(" + inner + "))", nil

	case callNode:
		if node.name == "empty" && len(node.args) == 1 {
			arg, err := compileNodeToSQL(node.args[0], dialect, auth, param, textExtractor)
			if err != nil {
				return "", err
			}
			return "(" + arg + " IS NULL OR " + arg + " = '')", nil
		}
		return "", fmt.Errorf("%w: dynamic function %q cannot be pushed to SQL", ErrInvalidExpression, node.name)

	default:
		return "", fmt.Errorf("%w: unknown expression node %T", ErrInvalidExpression, n)
	}
}
