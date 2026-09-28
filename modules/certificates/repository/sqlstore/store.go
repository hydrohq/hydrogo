package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/certificates"
)

type Store struct {
	sectoral.Store[certificates.Certificate]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[certificates.Certificate]{Dialect: dialect, Table: "sectoral_certificates"}}
}
