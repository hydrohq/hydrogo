package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/pricing"
)

type Store struct{ sectoral.Store[pricing.Plan] }

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[pricing.Plan]{Dialect: dialect, Table: "sectoral_pricing"}}
}
