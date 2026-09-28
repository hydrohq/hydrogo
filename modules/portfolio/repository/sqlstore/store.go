package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/portfolio"
)

type Store struct {
	sectoral.Store[portfolio.Project]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[portfolio.Project]{Dialect: dialect, Table: "sectoral_portfolio"}}
}
