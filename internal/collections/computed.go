package collections

import (
	"math/big"
	"strconv"
)

// validateComputed checks a computed field: a number field with fixed
// decimals whose expression only reads number fields declared before it.
func (s Schema) validateComputed(field Field) error {
	if field.Kind != Number || field.Decimals == nil || field.Required || field.Unique || field.Localized || field.Sequence != nil {
		return ErrInvalidSchema
	}
	expr, err := ParseExpr(field.Computed)
	if err != nil {
		return ErrInvalidSchema
	}
	earlier := map[string]bool{}
	for _, other := range s.Fields {
		if other.ID == field.ID {
			break
		}
		if other.Kind == Number {
			earlier[other.ID] = true
		}
	}
	for _, name := range expr.Fields() {
		if !earlier[name] {
			return ErrInvalidSchema
		}
	}
	return nil
}

// applyComputed fills the computed fields of a validated record, in
// declaration order so a field can use the ones before it.
func (s Schema) applyComputed(clean map[string]any) error {
	for _, field := range s.Fields {
		if field.Computed == "" {
			continue
		}
		expr, err := ParseExpr(field.Computed)
		if err != nil {
			return ErrInvalidSchema
		}
		result, err := expr.Eval(func(name string) *big.Rat { return decimalOf(clean[name]) })
		if err != nil {
			return fieldError(field.ID)
		}
		rounded := RoundHalfUp(result, *field.Decimals)
		value, err := strconv.ParseFloat(rounded.FloatString(*field.Decimals), 64)
		if err != nil || !fitsDecimals(value, *field.Decimals) {
			return fieldError(field.ID)
		}
		clean[field.ID] = value
	}
	return nil
}

// decimalOf converts a stored number to an exact decimal through its shortest
// text form, so 0.1 is one tenth and not the nearest binary fraction.
func decimalOf(value any) *big.Rat {
	number, ok := numeric(value)
	if !ok {
		return nil
	}
	rat, ok := new(big.Rat).SetString(strconv.FormatFloat(number, 'f', -1, 64))
	if !ok {
		return nil
	}
	return rat
}
