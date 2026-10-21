package collections

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

// Rollup totals, on a parent record, the records of another type that point
// at it. A "sales_orders" field can sum the line_total of its "order_lines"
// (those whose "order" relation holds the order's id); with no Sum it counts
// them. The server keeps the value current in the same transaction as every
// change to a line (create, update, delete, including a line moved to another
// parent), so it is never stale and never written by clients.
//
// A line cannot be changed while its parent is locked by a readonly rule
// (a posted invoice), because that would change the parent's total.
//
// Limits: at most MaxRollupRecords children per parent; MySQL's default
// isolation can leave a total stale when two lines of one parent are written
// at the same moment (the next change to a line corrects it).
type Rollup struct {
	From string `json:"from"`
	Via  string `json:"via"`
	Sum  string `json:"sum,omitempty"`
}

// MaxRollupRecords bounds how many child records one total may cover.
const MaxRollupRecords = 20000

func (r *Rollup) validate() error {
	if !identifier.MatchString(r.From) || !identifier.MatchString(r.Via) || (r.Sum != "" && !identifier.MatchString(r.Sum)) {
		return ErrInvalidSchema
	}
	return nil
}

func (s Schema) rollupFields() []Field {
	var fields []Field
	for _, field := range s.Fields {
		if field.Rollup != nil {
			fields = append(fields, field)
		}
	}
	return fields
}

// carryRollups puts the stored totals (0 for a new record) into a record
// about to be saved and recalculates the computed fields that read them.
func (s Schema) carryRollups(clean map[string]any, stored *Record) error {
	fields := s.rollupFields()
	if len(fields) == 0 {
		return nil
	}
	for _, field := range fields {
		clean[field.ID] = 0.0
		if stored != nil {
			if value, ok := stored.Data[field.ID]; ok {
				clean[field.ID] = value
			}
		}
	}
	return s.applyComputed(clean)
}

// withRollups runs a change of one child record (id empty for a create) and
// then recalculates the totals of the parents it touched, all in one
// transaction. The parents are those the record pointed at before and after.
func (s Service) withRollups(ctx context.Context, child Schema, id string, change func(Service) (Record, error)) (Record, error) {
	var result Record
	err := s.InTransaction(ctx, func(tx Service) error {
		before := map[string]map[string]bool{} // parent type -> parent ids
		note := func(data map[string]any, parents map[string]map[string]bool) error {
			for _, parentType := range child.RollupParents {
				parent, err := tx.Repository.GetSchema(ctx, parentType)
				if err != nil {
					return err
				}
				for _, field := range parent.rollupFields() {
					if field.Rollup.From != child.ID {
						continue
					}
					if parentID, _ := data[field.Rollup.Via].(string); parentID != "" {
						if parents[parentType] == nil {
							parents[parentType] = map[string]bool{}
						}
						parents[parentType][parentID] = true
					}
				}
			}
			return nil
		}
		if id != "" {
			stored, err := tx.Repository.GetRecord(ctx, child.ID, id)
			if err != nil {
				return err
			}
			if err := note(stored.Data, before); err != nil {
				return err
			}
		}
		record, err := change(tx)
		if err != nil {
			return err
		}
		result = record
		if record.ID != "" {
			if saved, err := tx.Repository.GetRecord(ctx, child.ID, record.ID); err == nil {
				if err := note(saved.Data, before); err != nil {
					return err
				}
			}
		}
		for parentType, ids := range before {
			parent, err := tx.Repository.GetSchema(ctx, parentType)
			if err != nil {
				return err
			}
			for parentID := range ids {
				if err := tx.recomputeRollups(ctx, parent, parentID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return result, err
}

// recomputeRollups totals every rollup field of one parent record from its
// current children and saves the parent (and the computed fields that use the
// totals). A parent that a readonly rule locks refuses the change.
func (s Service) recomputeRollups(ctx context.Context, parent Schema, parentID string) error {
	stored, err := s.Repository.GetRecord(ctx, parent.ID, parentID)
	if err != nil {
		return err
	}
	if err := parent.checkReadonly(stored.Data); err != nil {
		return err
	}
	// Writing the parent first takes its row lock, so concurrent changes to
	// its lines queue up and each total sees the earlier ones (PostgreSQL).
	unique := parent.uniqueFields(stored.Data)
	if _, err := s.Repository.SaveRecord(ctx, parent.ID, parentID, stored.Data, unique, 0); err != nil {
		return err
	}
	data := make(map[string]any, len(stored.Data))
	for key, value := range stored.Data {
		data[key] = value
	}
	for _, field := range parent.rollupFields() {
		total, err := s.totalChildren(ctx, field, parentID)
		if err != nil {
			return err
		}
		value, err := strconv.ParseFloat(RoundHalfUp(total, *field.Decimals).FloatString(*field.Decimals), 64)
		if err != nil || !fitsDecimals(value, *field.Decimals) {
			return fieldError(field.ID)
		}
		data[field.ID] = value
	}
	if err := parent.applyComputed(data); err != nil {
		return err
	}
	if err := parent.checkAsserts(data); err != nil {
		return err
	}
	_, err = s.Repository.SaveRecord(ctx, parent.ID, parentID, data, unique, 0)
	return err
}

func (s Service) totalChildren(ctx context.Context, field Field, parentID string) (*big.Rat, error) {
	total := new(big.Rat)
	query := Query{Limit: 100, Where: []Condition{{Field: field.Rollup.Via, Op: Equals, Value: parentID}}}
	seen := 0
	for {
		records, err := s.QueryRecords(ctx, field.Rollup.From, query)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if seen++; seen > MaxRollupRecords {
				return nil, fmt.Errorf("%w: more than %d records to total", ErrInvalidRecord, MaxRollupRecords)
			}
			if field.Rollup.Sum == "" {
				total.Add(total, big.NewRat(1, 1))
			} else if number, ok := ExactNumber(record.Data[field.Rollup.Sum]); ok {
				total.Add(total, number)
			}
		}
		if len(records) < query.Limit {
			return total, nil
		}
		cursor, err := DecodeCursor(NextCursor(query, records[len(records)-1]))
		if err != nil {
			return nil, errors.New("rollup paging failed")
		}
		query.After = &cursor
	}
}

// uniqueFields lists the unique or numbered fields present in data, as the
// store needs them to keep its unique index in step with a save.
func (s Schema) uniqueFields(data map[string]any) []Field {
	var unique []Field
	for _, field := range s.Fields {
		if _, present := data[field.ID]; present && (field.Unique || field.Sequence != nil) {
			unique = append(unique, field)
		}
	}
	return unique
}
