package auth_test

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
)

func TestReservedServiceEmailDomain(t *testing.T) {
	db, dialect := setupTestDB(t)
	service := auth.NewAuthService(authstore.NewUserStore(db, dialect), authstore.NewSessionStore(db, dialect), auth.NewMobileAuthService(db, dialect), auth.DefaultArgon2Params)
	ctx := context.Background()
	const password = "correct horse battery staple"

	// People cannot register an address in the reserved domain (any casing)...
	for _, email := range []string{"service+x@hydrogo.invalid", "Boss@HYDROGO.INVALID"} {
		if _, err := service.CreateUser(ctx, email, password, auth.RoleEditor); err == nil {
			t.Fatalf("CreateUser(%q) must be rejected", email)
		}
	}
	// ...but the service-account path can.
	if _, err := service.CreateServiceUser(ctx, "service+seed@hydrogo.invalid", password, auth.RoleViewer); err != nil {
		t.Fatalf("CreateServiceUser: %v", err)
	}
	// A normal user cannot switch to a reserved address either.
	user, err := service.CreateUser(ctx, "bob@example.test", password, auth.RoleEditor)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangeCredentials(ctx, user.ID, password, "hidden@hydrogo.invalid", "", nil); err == nil {
		t.Fatal("ChangeCredentials to a reserved address must be rejected")
	}
}
