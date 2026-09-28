package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/team"
)

type Store struct{ sectoral.Store[team.Member] }

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[team.Member]{Dialect: dialect, Table: "sectoral_team"}}
}
