package outbox

import (
	"errors"
	"time"
)

var ErrLeaseInvalid = errors.New("outbox lease is invalid")

type Event struct {
	ID             string
	Topic          string
	AggregateID    string
	PayloadVersion int
	Attempts       int
	AvailableAt    time.Time
	LastError      string
}
