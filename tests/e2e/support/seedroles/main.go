// Command seedroles creates deterministic editor/viewer accounts for the
// isolated Admin Studio browser test database only.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func main() {
	dsn := flag.String("database-dsn", "", "SQLite database path")
	flag.Parse()
	if *dsn == "" {
		log.Fatal("--database-dsn is required")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	users := sqlstore.NewUserStore(db, database.SQLite)
	service := auth.NewAuthService(users, nil, nil, auth.DefaultArgon2Params)
	for _, account := range []struct {
		email string
		role  auth.Role
	}{
		{email: "editor@example.test", role: auth.RoleEditor},
		{email: "viewer@example.test", role: auth.RoleViewer},
	} {
		if _, err := service.CreateUser(ctx, account.email, "Lime-Train7!Harbor", account.role); err != nil {
			log.Fatal(err)
		}
	}
}
