package collections

import (
	"math/big"
	"testing"
)

func evalExpr(t *testing.T, source string, vars map[string]string) (string, error) {
	t.Helper()
	expr, err := ParseExpr(source)
	if err != nil {
		return "", err
	}
	result, err := expr.EvalNumber(func(name string) Value {
		if text, ok := vars[name]; ok {
			value, _ := new(big.Rat).SetString(text)
			return value
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return result.FloatString(6), nil
}

func TestExpressionsUseExactDecimalArithmetic(t *testing.T) {
	cases := []struct {
		source string
		vars   map[string]string
		want   string
	}{
		{"0.1 + 0.2", nil, "0.300000"},
		{"quantity * unit_price", map[string]string{"quantity": "3", "unit_price": "19.99"}, "59.970000"},
		{"subtotal * (1 + tax_rate)", map[string]string{"subtotal": "100", "tax_rate": "0.18"}, "118.000000"},
		{"-a + 10", map[string]string{"a": "4"}, "6.000000"},
		{"round(10 / 3, 2)", nil, "3.330000"},
		{"round(2.675, 2)", nil, "2.680000"},
		{"round(-2.675, 2)", nil, "-2.680000"},
		{"max(a, b) - min(a, b)", map[string]string{"a": "5", "b": "8"}, "3.000000"},
		{"abs(0 - 7)", nil, "7.000000"},
		{"missing + 1", nil, "1.000000"},
	}
	for _, tc := range cases {
		got, err := evalExpr(t, tc.source, tc.vars)
		if err != nil || got != tc.want {
			t.Errorf("%s = %q, %v; want %s", tc.source, got, err, tc.want)
		}
	}
}

func TestExpressionsRejectBadInput(t *testing.T) {
	for _, source := range []string{"", "1 +", "a b", "(1", "1)", "foo(1)", "round(1)", "min(1,2,3)", "1 $ 2", "a.b", "1..2", "a; b", "'open"} {
		if _, err := ParseExpr(source); err == nil {
			t.Errorf("%q accepted", source)
		}
	}
	if _, err := evalExpr(t, "1 / 0", nil); err == nil {
		t.Error("division by zero accepted")
	}
	long := "1"
	for i := 0; i < 200; i++ {
		long += "+1"
	}
	if _, err := ParseExpr(long); err == nil {
		t.Error("oversized expression accepted")
	}
	deep := ""
	for i := 0; i < 40; i++ {
		deep += "("
	}
	if _, err := ParseExpr(deep + "1" + string(make([]byte, 0))); err == nil {
		t.Error("deep or unbalanced expression accepted")
	}
}

func invoiceSchema() Schema {
	two := 2
	return Schema{ID: "invoices", Label: "Invoices", Fields: []Field{
		{ID: "quantity", Label: "Qty", Kind: Number},
		{ID: "unit_price", Label: "Price", Kind: Number, Decimals: &two},
		{ID: "subtotal", Label: "Subtotal", Kind: Number, Decimals: &two, Computed: "quantity * unit_price"},
		{ID: "tax", Label: "Tax", Kind: Number, Decimals: &two, Computed: "subtotal * 0.18"},
		{ID: "total", Label: "Total", Kind: Number, Decimals: &two, Computed: "subtotal + tax"},
	}}
}

func TestComputedFieldsAreCalculatedAndClientValuesIgnored(t *testing.T) {
	schema := invoiceSchema()
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	record, err := schema.ValidateRecord(map[string]any{"quantity": 3.0, "unit_price": 19.99, "total": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if record["subtotal"] != 59.97 || record["tax"] != 10.79 || record["total"] != 70.76 {
		t.Fatalf("computed values = %v", record)
	}
}

func TestComputedFieldSchemaRules(t *testing.T) {
	two := 2
	number := func(id string) Field { return Field{ID: id, Label: id, Kind: Number} }
	cases := map[string][]Field{
		"no decimals":         {number("a"), {ID: "c", Label: "c", Kind: Number, Computed: "a"}},
		"forward reference":   {{ID: "c", Label: "c", Kind: Number, Decimals: &two, Computed: "a"}, number("a")},
		"self reference":      {{ID: "c", Label: "c", Kind: Number, Decimals: &two, Computed: "c"}},
		"unknown field":       {{ID: "c", Label: "c", Kind: Number, Decimals: &two, Computed: "nope"}},
		"text field":          {number("a"), {ID: "c", Label: "c", Kind: Text, Computed: "a"}},
		"syntax error":        {number("a"), {ID: "c", Label: "c", Kind: Number, Decimals: &two, Computed: "a +"}},
		"required & computed": {number("a"), {ID: "c", Label: "c", Kind: Number, Decimals: &two, Required: true, Computed: "a"}},
	}
	for name, fields := range cases {
		if (Schema{ID: "x", Label: "X", Fields: fields}).Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestComputedOverflowAndDivisionFailTheSave(t *testing.T) {
	two := 2
	schema := Schema{ID: "x", Label: "X", Fields: []Field{
		{ID: "a", Label: "a", Kind: Number},
		{ID: "b", Label: "b", Kind: Number},
		{ID: "ratio", Label: "r", Kind: Number, Decimals: &two, Computed: "a / b"},
	}}
	if _, err := schema.ValidateRecord(map[string]any{"a": 1.0, "b": 0.0}); err == nil {
		t.Error("division by zero saved")
	}
	if _, err := schema.ValidateRecord(map[string]any{"a": 1e12, "b": 1e-3}); err == nil {
		t.Error("overflowing result saved")
	}
}

func evalCondition(t *testing.T, source string, vars map[string]Value) (bool, error) {
	t.Helper()
	expr, err := ParseExpr(source)
	if err != nil {
		return false, err
	}
	return expr.EvalBool(func(name string) Value { return vars[name] })
}

func TestConditions(t *testing.T) {
	num := func(text string) Value { value, _ := new(big.Rat).SetString(text); return value }
	vars := map[string]Value{"status": "posted", "total": num("118.5"), "paid": true, "note": ""}
	cases := map[string]bool{
		"status == 'posted'":                 true,
		"status != 'posted'":                 false,
		"total > 100 && paid":                true,
		"total <= 100 || status == 'void'":   false,
		"!paid":                              false,
		"total * 2 == 237":                   true,
		"empty(note) && !empty(status)":      true,
		"missing == ''":                      true,
		"missing == 0":                       true,
		"(total > 1) == true":                true,
		"status == 'a' || total > 1 && paid": true,
		"false && 1 / 0 > 1":                 false, // right side is not evaluated
		"status < 'q'":                       true,
		"status >= 'p'":                      true,
	}
	for source, want := range cases {
		got, err := evalCondition(t, source, vars)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", source, got, err, want)
		}
	}
	for _, source := range []string{"total > 'x'", "status + 1", "paid > true", "total", "!total", "paid && 1"} {
		if _, err := evalCondition(t, source, vars); err == nil {
			t.Errorf("%q evaluated without error", source)
		}
	}
}

func TestRuleAndStateSchemaValidation(t *testing.T) {
	good := func() Schema {
		return Schema{ID: "x", Label: "X", Fields: []Field{
			{ID: "status", Label: "S", Kind: Select, Options: []string{"a", "b"}},
			{ID: "n", Label: "N", Kind: Number},
		}}
	}
	ok := good()
	ok.Rules = []Rule{{Assert: "n > 0"}, {When: "status == 'b'", Readonly: true}}
	ok.States = &StateMachine{Field: "status", Initial: "a", Transitions: []Transition{{From: "a", To: "b", When: "n > 0", Permission: "x.approve"}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Schema){
		"rule without assert or readonly": func(s *Schema) { s.Rules = []Rule{{When: "n > 0"}} },
		"readonly without when":           func(s *Schema) { s.Rules = []Rule{{Readonly: true}} },
		"readonly with assert":            func(s *Schema) { s.Rules = []Rule{{When: "n > 0", Readonly: true, Assert: "n > 1"}} },
		"unknown field in assert":         func(s *Schema) { s.Rules = []Rule{{Assert: "nope > 0"}} },
		"syntax error":                    func(s *Schema) { s.Rules = []Rule{{Assert: "n >"}} },
		"state field not a select": func(s *Schema) {
			s.States = &StateMachine{Field: "n", Initial: "a", Transitions: []Transition{{From: "a", To: "b"}}}
		},
		"unknown initial": func(s *Schema) {
			s.States = &StateMachine{Field: "status", Initial: "z", Transitions: []Transition{{From: "a", To: "b"}}}
		},
		"unknown target": func(s *Schema) {
			s.States = &StateMachine{Field: "status", Initial: "a", Transitions: []Transition{{From: "a", To: "z"}}}
		},
		"no transitions": func(s *Schema) { s.States = &StateMachine{Field: "status", Initial: "a"} },
		"duplicate transition": func(s *Schema) {
			s.States = &StateMachine{Field: "status", Initial: "a", Transitions: []Transition{{From: "a", To: "b"}, {From: "a", To: "b"}}}
		},
		"bad permission": func(s *Schema) {
			s.States = &StateMachine{Field: "status", Initial: "a", Transitions: []Transition{{From: "a", To: "b", Permission: "Not Valid"}}}
		},
	}
	for name, mutate := range mutations {
		schema := good()
		mutate(&schema)
		if schema.Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
