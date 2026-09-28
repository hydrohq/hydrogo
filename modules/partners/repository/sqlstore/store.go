package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/partners"
)

type Store struct {
	sectoral.Store[partners.Partner]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[partners.Partner]{Dialect: dialect, Table: "sectoral_partners"}}
}
