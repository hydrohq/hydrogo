package sqlstore

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
)

func TestMySQLStaticStoreSupportsQueriesAndRevisions(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	runStaticStoreContract(t, db, database.MySQL)
	runQueryContract(t, db, database.MySQL)
	runRevisionContract(t, db, database.MySQL)
}
