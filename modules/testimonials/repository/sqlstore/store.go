package sqlstore

import (
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/testimonials"
)

type Store struct {
	sectoral.Store[testimonials.Testimonial]
}

func New(dialect database.Dialect) Store {
	return Store{Store: sectoral.Store[testimonials.Testimonial]{Dialect: dialect, Table: "sectoral_testimonials"}}
}
