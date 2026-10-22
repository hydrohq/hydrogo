package sqlstore

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
)

func TestMySQLStaticStoreSupportsQueriesRevisionsAndTranslations(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	runStaticStoreContract(t, db, database.MySQL)
	runQueryContract(t, db, database.MySQL)
	runReferenceContract(t, db, database.MySQL)
	runRevisionContract(t, db, database.MySQL)
	runTranslationContract(t, db, database.MySQL)
	runFilterContract(t, db, database.MySQL)
	runSequenceContract(t, db, database.MySQL)
	runRulesContract(t, db, database.MySQL)
	runRollupContract(t, db, database.MySQL)
	runRollupConcurrencyContract(t, db, database.MySQL)
}
