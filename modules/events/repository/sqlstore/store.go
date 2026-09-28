package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/events"
)

type Store struct{ sectoral.Store[events.Event] }

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[events.Event]{Dialect: dialect, Table: "sectoral_events"}}
}
