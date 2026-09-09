package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func requireUserStore(store *UserStore) error {
	if store == nil {
		return database.ErrDatabaseRequired
	}
	return nil
}

func requireUserDB(store *UserStore) error {
	if err := requireUserStore(store); err != nil {
		return err
	}
	if store.db == nil {
		return database.ErrDatabaseRequired
	}
	return nil
}

func requireSessionStore(store *SessionStore) error {
	if store == nil || store.db == nil {
		return database.ErrDatabaseRequired
	}
	return nil
}

func requireAuthExecutor(executor database.Executor) error {
	return database.RequireExecutor(executor)
}
