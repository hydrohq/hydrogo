package auth

import (
	"context"
	"errors"
	"net/mail"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type User struct {
	ID                  string          `json:"id"`
	Email               string          `json:"email"`
	PasswordHash        string          `json:"-"`
	Role                Role            `json:"role"`
	Active              bool            `json:"active"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	PermissionOverrides map[string]bool `json:"-"`
}

// UserAdminRepository contains the administrative account operations used by
// the Admin Studio users surface. It is deliberately separate from the login
// repository so member identities cannot accidentally implement this boundary.
type UserAdminRepository interface {
	List(ctx context.Context) ([]*User, error)
	UpdateEmail(ctx context.Context, tx database.Executor, userID, email string) error
	UpdateRole(ctx context.Context, userID string, role Role) error
}

func NormalizeEmail(email string) string {
	return canonical.Email(email)
}

func ValidateEmail(email string) bool {
	email = NormalizeEmail(email)
	if email == "" || len(email) > 320 {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Count(ctx context.Context) (int64, error)
	SetActive(ctx context.Context, id string, active bool) error
	UpdatePassword(ctx context.Context, tx database.Executor, userID, passwordHash string) error
	UpdateEmail(ctx context.Context, tx database.Executor, userID, email string) error
}
