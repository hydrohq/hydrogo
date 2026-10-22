package collections

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Sequence numbers records without gaps: each new record of a type gets the
// next number, formatted as Prefix followed by the number left-padded with
// zeros to Pad digits. The prefix may contain {yyyy}, {yy} and {mm}; the
// counter restarts for every distinct expanded prefix, so "INV-{yyyy}-"
// numbers invoices per year. Numbers are taken in the same transaction as the
// record, so a failed or rolled-back save does not use one up.
type Sequence struct {
	Prefix string `json:"prefix,omitempty"`
	Pad    int    `json:"pad,omitempty"`
}

var sequencePrefix = regexp.MustCompile(`^[A-Za-z0-9_./{}-]{0,32}$`)

func (q Sequence) validate() error {
	if !sequencePrefix.MatchString(q.Prefix) || q.Pad < 0 || q.Pad > 12 {
		return ErrInvalidSchema
	}
	rest := strings.NewReplacer("{yyyy}", "", "{yy}", "", "{mm}", "").Replace(q.Prefix)
	if strings.ContainsAny(rest, "{}") {
		return ErrInvalidSchema
	}
	return nil
}

// Scope returns the counter key for a moment: the prefix with its date tokens
// expanded.
func (q Sequence) Scope(now time.Time) string {
	now = now.UTC()
	return strings.NewReplacer("{yyyy}", now.Format("2006"), "{yy}", now.Format("06"), "{mm}", now.Format("01")).Replace(q.Prefix)
}

// Format renders number n within scope.
func (q Sequence) Format(scope string, n int64) string {
	return fmt.Sprintf("%s%0*d", scope, q.Pad, n)
}

// SequenceRepository hands out sequence numbers. NextSequence must be called
// inside the transaction that saves the record (see Transactional).
type SequenceRepository interface {
	NextSequence(ctx context.Context, collectionID, fieldID, scope string) (int64, error)
}

func (s Schema) sequenceFields() []Field {
	var fields []Field
	for _, field := range s.Fields {
		if field.Sequence != nil {
			fields = append(fields, field)
		}
	}
	return fields
}
