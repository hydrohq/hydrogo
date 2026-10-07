// Command seedinvite issues a real, SQL-backed invitation for browser tests.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func main() {
	dsn := flag.String("database-dsn", "", "SQLite database path")
	email := flag.String("email", "", "invited email")
	flag.Parse()
	if *dsn == "" || *email == "" {
		log.Fatal("--database-dsn and --email are required")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	users := sqlstore.NewUserStore(db, database.SQLite)
	service := auth.InvitationService{Users: users, Invites: users, ArgonParams: auth.DefaultArgon2Params}
	token, err := service.Issue(ctx, db, *email, auth.RoleEditor, time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(token)
}
