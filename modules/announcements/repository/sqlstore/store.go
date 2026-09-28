package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/announcements"
)

type Store struct {
	sectoral.Store[announcements.Announcement]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[announcements.Announcement]{Dialect: dialect, Table: "sectoral_announcements"}}
}
