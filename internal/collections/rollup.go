package collections

import (
	"context"
	"errors"
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
// The totals are kept incrementally (each write adds or subtracts the
// record's contribution), so they cost two parent reads per write however
// many lines there are. Concurrent writes to one parent are serialised by the
// parent's version: the loser re-reads it and retries (three times, then
// ErrVersionConflict). Changing lines directly in the database leaves the
// total as it was.
type Rollup struct {
	From string `json:"from"`
	Via  string `json:"via"`
	Sum  string `json:"sum,omitempty"`
}

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
// then adjusts the totals of the parents it touched, all in one transaction.
// The adjustment is the difference between what the record contributed before
// and after (a create adds, a delete subtracts, a move changes two parents),
// so a write costs two parent reads however many lines a parent has.
func (s Service) withRollups(ctx context.Context, child Schema, id string, change func(Service) (Record, error)) (Record, error) {
	var result Record
	err := s.InTransaction(ctx, func(tx Service) error {
		deltas := map[rollupKey]*big.Rat{}
		if id != "" {
			stored, err := tx.Repository.GetRecord(ctx, child.ID, id)
			if err != nil {
				return err
			}
			if err := tx.addContribution(ctx, child, stored.Data, -1, deltas); err != nil {
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
				if err := tx.addContribution(ctx, child, saved.Data, 1, deltas); err != nil {
					return err
				}
			}
		}
		parents := map[rollupParent]bool{}
		for key := range deltas {
			parents[key.rollupParent] = true
		}
		for parent := range parents {
			if err := tx.adjustParent(ctx, parent, deltas); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

type rollupParent struct{ typeID, id string }

type rollupKey struct {
	rollupParent
	field string
}

// addContribution adds sign times what a child record contributes to each
// parent total it feeds. Parents are recorded even when the amount is zero, so
// a locked parent still refuses the change.
func (s Service) addContribution(ctx context.Context, child Schema, data map[string]any, sign int64, deltas map[rollupKey]*big.Rat) error {
	for _, parentType := range child.RollupParents {
		parent, err := s.Repository.GetSchema(ctx, parentType)
		if err != nil {
			return err
		}
		for _, field := range parent.rollupFields() {
			if field.Rollup.From != child.ID {
				continue
			}
			parentID, _ := data[field.Rollup.Via].(string)
			if parentID == "" {
				continue
			}
			amount := big.NewRat(1, 1)
			if field.Rollup.Sum != "" {
				number, ok := ExactNumber(data[field.Rollup.Sum])
				if !ok {
					number = new(big.Rat)
				}
				amount = number
			}
			key := rollupKey{rollupParent{parentType, parentID}, field.ID}
			if deltas[key] == nil {
				deltas[key] = new(big.Rat)
			}
			deltas[key].Add(deltas[key], new(big.Rat).Mul(amount, big.NewRat(sign, 1)))
		}
	}
	return nil
}

// rollupAttempts bounds the retries when a parent changed under us.
const rollupAttempts = 3

// adjustParent applies the deltas to one parent record and recalculates the
// computed fields that read the totals. A parent that a readonly rule locks
// refuses the change. The save checks the parent's version, so two writers
// never overwrite each other's totals; on a conflict the parent is read again.
func (s Service) adjustParent(ctx context.Context, target rollupParent, deltas map[rollupKey]*big.Rat) error {
	schema, err := s.Repository.GetSchema(ctx, target.typeID)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < rollupAttempts; attempt++ {
		stored, err := s.Repository.GetRecord(ctx, target.typeID, target.id)
		if err != nil {
			return err
		}
		if err := schema.checkReadonly(stored.Data); err != nil {
			return err
		}
		data := make(map[string]any, len(stored.Data))
		for key, value := range stored.Data {
			data[key] = value
		}
		changed := false
		for _, field := range schema.rollupFields() {
			delta, ok := deltas[rollupKey{target, field.ID}]
			if !ok || delta.Sign() == 0 {
				continue
			}
			current, _ := ExactNumber(data[field.ID])
			if current == nil {
				current = new(big.Rat)
			}
			total := new(big.Rat).Add(current, delta)
			value, err := strconv.ParseFloat(RoundHalfUp(total, *field.Decimals).FloatString(*field.Decimals), 64)
			if err != nil || !fitsDecimals(value, *field.Decimals) {
				return fieldError(field.ID)
			}
			data[field.ID] = value
			changed = true
		}
		if !changed {
			return nil
		}
		if err := schema.applyComputed(data); err != nil {
			return err
		}
		if err := schema.checkAsserts(data); err != nil {
			return err
		}
		_, err = s.Repository.SaveRecord(ctx, target.typeID, target.id, data, schema.uniqueFields(data), stored.Version)
		if errors.Is(err, ErrVersionConflict) {
			continue
		}
		return err
	}
	return ErrVersionConflict
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
