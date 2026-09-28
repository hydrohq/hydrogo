package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/faq"
)

type Store struct{ sectoral.Store[faq.Question] }

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[faq.Question]{Dialect: dialect, Table: "sectoral_faq"}}
}
