package collections

import (
	"errors"
	"testing"
)

func querySchema() Schema {
	return Schema{ID: "faq", Label: "FAQ", Fields: []Field{
		{ID: "question", Label: "Question", Kind: Text},
		{ID: "order", Label: "Order", Kind: Number},
		{ID: "featured", Label: "Featured", Kind: Boolean},
		{ID: "tags", Label: "Tags", Kind: MultiSelect, Options: []string{"a", "b"}},
	}}
}

func TestValidateQueryConvertsValuesAndRejectsUnknownFields(t *testing.T) {
	schema := querySchema()
	checked, err := schema.ValidateQuery(Query{Limit: 10,
		Where: []Condition{{Field: "order", Op: Equals, Value: " 3 "}, {Field: "featured", Op: Equals, Value: "true"}},
		AnyOf: []Condition{{Field: "question", Op: Contains, Value: "ship"}},
		Sort:  &Sort{Field: "order", Desc: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked.Where[0].Typed != 3.0 || checked.Where[1].Typed != true || checked.AnyOf[0].Typed != "ship" || checked.Sort.Kind != Number {
		t.Fatalf("checked=%+v", checked)
	}
	if checked, err := schema.ValidateQuery(Query{Limit: 1, Sort: &Sort{Field: SortCreatedAt}}); err != nil || checked.Sort.Kind != "" {
		t.Fatalf("system sort: %+v %v", checked, err)
	}
	bad := map[string]Query{
		"unknown field":       {Limit: 1, Where: []Condition{{Field: "missing", Op: Equals, Value: "x"}}},
		"injection in field":  {Limit: 1, Where: []Condition{{Field: "question') OR 1=1 --", Op: Equals, Value: "x"}}},
		"json path in field":  {Limit: 1, AnyOf: []Condition{{Field: "question.secret", Op: Contains, Value: "x"}}},
		"injection in sort":   {Limit: 1, Sort: &Sort{Field: "question; DROP TABLE hydro_collections"}},
		"unknown operator":    {Limit: 1, Where: []Condition{{Field: "question", Op: "like", Value: "x"}}},
		"contains on number":  {Limit: 1, Where: []Condition{{Field: "order", Op: Contains, Value: "1"}}},
		"equals on an array":  {Limit: 1, Where: []Condition{{Field: "tags", Op: Equals, Value: "a"}}},
		"not a number":        {Limit: 1, Where: []Condition{{Field: "order", Op: Equals, Value: "1 OR 1=1"}}},
		"not a boolean":       {Limit: 1, Where: []Condition{{Field: "featured", Op: Equals, Value: "yes please"}}},
		"limit out of range":  {Limit: 0},
		"offset out of range": {Limit: 1, Offset: -1},
		"value too long":      {Limit: 1, Where: []Condition{{Field: "question", Op: Equals, Value: string(make([]byte, MaxQueryValueBytes+1))}}},
	}
	for name, query := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := schema.ValidateQuery(query); !errors.Is(err, ErrInvalidQuery) {
				t.Fatalf("err=%v, want ErrInvalidQuery", err)
			}
		})
	}
	tooMany := Query{Limit: 1}
	for range MaxQueryConditions + 1 {
		tooMany.Where = append(tooMany.Where, Condition{Field: "question", Op: Equals, Value: "x"})
	}
	if _, err := schema.ValidateQuery(tooMany); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("too many conditions: %v", err)
	}
}

func TestParseSort(t *testing.T) {
	if ParseSort("") != nil {
		t.Fatal("empty sort")
	}
	if sort := ParseSort("-order"); sort.Field != "order" || !sort.Desc {
		t.Fatalf("sort=%+v", sort)
	}
	if sort := ParseSort("question"); sort.Field != "question" || sort.Desc {
		t.Fatalf("sort=%+v", sort)
	}
}
