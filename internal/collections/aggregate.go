package collections

import (
	"fmt"
	"net/url"
	"strings"
)

type AggregateFunc string

const (
	AggCount AggregateFunc = "count"
	AggSum   AggregateFunc = "sum"
	AggAvg   AggregateFunc = "avg"
	AggMin   AggregateFunc = "min"
	AggMax   AggregateFunc = "max"
)

type AggregateSpec struct {
	Func  AggregateFunc `json:"func"`
	Field string        `json:"field,omitempty"`
	Alias string        `json:"alias"`
}

type AggregateQuery struct {
	Where       []Condition
	AnyOf       []Condition
	AccessRule  string
	AuthContext AuthContext
	Aggregates  []AggregateSpec
	GroupBy     []string
}

// ParseAggregateQuery parses ?aggregate=... and ?group_by=... from URL query parameters.
func ParseAggregateQuery(values url.Values) (AggregateQuery, bool, error) {
	rawAgg := strings.TrimSpace(values.Get("aggregate"))
	if rawAgg == "" {
		return AggregateQuery{}, false, nil
	}

	var specs []AggregateSpec
	tokens := strings.Split(rawAgg, ",")
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if tok == "count" || tok == "count(*)" || tok == "count(1)" {
			specs = append(specs, AggregateSpec{
				Func:  AggCount,
				Alias: "count",
			})
			continue
		}
		open := strings.IndexByte(tok, '(')
		close := strings.IndexByte(tok, ')')
		if open <= 0 || close != len(tok)-1 || close <= open+1 {
			return AggregateQuery{}, false, fmt.Errorf("%w: invalid aggregate specification %q", ErrInvalidQuery, tok)
		}
		fnName := strings.ToLower(tok[:open])
		fieldName := tok[open+1 : close]
		if !identifier.MatchString(fieldName) {
			return AggregateQuery{}, false, fmt.Errorf("%w: invalid aggregate field %q", ErrInvalidQuery, fieldName)
		}
		var fn AggregateFunc
		switch fnName {
		case "sum":
			fn = AggSum
		case "avg":
			fn = AggAvg
		case "min":
			fn = AggMin
		case "max":
			fn = AggMax
		default:
			return AggregateQuery{}, false, fmt.Errorf("%w: unsupported aggregate function %q", ErrInvalidQuery, fnName)
		}
		specs = append(specs, AggregateSpec{
			Func:  fn,
			Field: fieldName,
			Alias: fnName + "_" + fieldName,
		})
	}
	if len(specs) == 0 {
		return AggregateQuery{}, false, nil
	}

	var groupBy []string
	if rawGroup := strings.TrimSpace(values.Get("group_by")); rawGroup != "" {
		for _, g := range strings.Split(rawGroup, ",") {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			if !identifier.MatchString(g) {
				return AggregateQuery{}, false, fmt.Errorf("%w: invalid group_by field %q", ErrInvalidQuery, g)
			}
			groupBy = append(groupBy, g)
		}
	}

	return AggregateQuery{
		Aggregates: specs,
		GroupBy:    groupBy,
	}, true, nil
}
