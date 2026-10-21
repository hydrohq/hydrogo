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

func TestValidateQueryAcceptsComparisonOperators(t *testing.T) {
	schema := Schema{ID: "deals", Label: "Deals", Fields: []Field{
		{ID: "stage", Label: "Stage", Kind: Select, Options: []string{"lead", "won"}},
		{ID: "amount", Label: "Amount", Kind: Number},
		{ID: "closes", Label: "Closes", Kind: DateTime},
		{ID: "company", Label: "Company", Kind: Relation, Relation: "companies"},
		{ID: "open", Label: "Open", Kind: Boolean},
		{ID: "tags", Label: "Tags", Kind: MultiSelect, Options: []string{"a"}},
	}}
	checked, err := schema.ValidateQuery(Query{Limit: 10, Where: []Condition{
		{Field: "amount", Op: GreaterOrEqual, Value: "100"},
		{Field: "closes", Op: LessThan, Value: "2026-10-01"},
		{Field: "stage", Op: In, Value: "lead, won"},
		{Field: "company", Op: NotEquals, Value: "c1"},
		{Field: "open", Op: IsNull, Value: "false"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if checked.Where[0].Typed != 100.0 || checked.Where[1].Typed != "2026-10-01T00:00:00Z" || checked.Where[4].Typed != false {
		t.Fatalf("checked=%+v", checked.Where)
	}
	if in, ok := checked.Where[2].Typed.([]any); !ok || len(in) != 2 || in[1] != "won" {
		t.Fatalf("in=%+v", checked.Where[2].Typed)
	}
	bad := map[string]Condition{
		"gt on text":           {Field: "stage", Op: GreaterThan, Value: "a"},
		"gt on boolean":        {Field: "open", Op: GreaterThan, Value: "true"},
		"bad date":             {Field: "closes", Op: LessThan, Value: "next week"},
		"bad number in list":   {Field: "amount", Op: In, Value: "1,x"},
		"null needs boolean":   {Field: "amount", Op: IsNull, Value: "maybe"},
		"null on multiselect":  {Field: "tags", Op: IsNull, Value: "true"},
		"ne on multiselect":    {Field: "tags", Op: NotEquals, Value: "a"},
		"in with 51 values":    {Field: "amount", Op: In, Value: "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51"},
		"unknown operator":     {Field: "amount", Op: "between", Value: "1"},
		"injection in in list": {Field: "stage", Op: In, Value: "a') OR 1=1 --,b"},
	}
	for name, condition := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := schema.ValidateQuery(Query{Limit: 1, Where: []Condition{condition}})
			// The injection case is valid text: it must stay a bound value.
			if name == "injection in in list" {
				if err != nil {
					t.Fatalf("a quote in a text value is data, got %v", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidQuery) {
				t.Fatalf("err=%v, want ErrInvalidQuery", err)
			}
		})
	}
}
