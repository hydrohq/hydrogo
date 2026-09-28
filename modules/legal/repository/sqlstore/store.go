package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/legal"
)

type Store struct{ sectoral.Store[legal.Document] }

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[legal.Document]{Dialect: dialect, Table: "sectoral_legal"}}
}
