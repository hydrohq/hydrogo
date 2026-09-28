package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/locations"
)

type Store struct {
	sectoral.Store[locations.Location]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[locations.Location]{Dialect: dialect, Table: "sectoral_locations"}}
}
