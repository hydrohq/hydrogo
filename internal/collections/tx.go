package collections

import (
	"context"
	"errors"
)

// ErrNoTransactions is returned when the repository cannot run several
// operations atomically.
var ErrNoTransactions = errors.New("repository does not support transactions")

// Transactional is implemented by repositories that can run several
// operations in one database transaction: either all of them are saved or none.
type Transactional interface {
	InTransaction(ctx context.Context, fn func(Repository) error) error
}

// InTransaction runs fn with a Service bound to one transaction. When fn
// returns an error everything it saved is rolled back.
func (s Service) InTransaction(ctx context.Context, fn func(Service) error) error {
	transactional, ok := s.Repository.(Transactional)
	if !ok {
		return ErrNoTransactions
	}
	return transactional.InTransaction(ctx, func(repository Repository) error {
		return fn(Service{Repository: repository})
	})
}
