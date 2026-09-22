package query

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func schema() map[string]Field {
	return map[string]Field{
		"status": {Name: "status", Operators: map[string]bool{"equals": true, "in": true}},
		"title":  {Name: "title", Operators: map[string]bool{"contains": true}},
		"age":    {Name: "age", Operators: map[string]bool{"greater_than": true}},
	}
}

func TestParseAndBuildSQLUsesAllowlistAndParameters(t *testing.T) {
	values := url.Values{
		"where[status][in]":      []string{"draft,published"},
		"where[title][contains]": []string{"a'b"},
		"sort":                   []string{"-status"},
		"limit":                  []string{"20"},
	}
	parsed, err := Parse(values, schema())
	if err != nil {
		t.Fatal(err)
	}
	built, err := parsed.BuildSQL(database.Postgres, schema())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(built.Where, "status IN (") || !strings.Contains(built.Where, "title LIKE $") || built.Order != "status DESC" {
		t.Fatalf("unexpected SQL: %#v", built)
	}
	if len(built.Args) != 3 {
		t.Fatalf("unexpected args: %#v", built.Args)
	}
	joinedArgs := strings.Join([]string{built.Args[0].(string), built.Args[1].(string), built.Args[2].(string)}, "|")
	if !strings.Contains(joinedArgs, "%a'b%") || !strings.Contains(joinedArgs, "draft") || !strings.Contains(joinedArgs, "published") {
		t.Fatalf("unexpected args: %#v", built.Args)
	}
}

func TestParseRejectsUnknownFieldsOperatorsAndBudgets(t *testing.T) {
	for _, values := range []url.Values{
		{"where[secret][equals]": []string{"x"}},
		{"where[status][contains]": []string{"x"}},
		{"where[status][equals]": []string{strings.Repeat("x", MaxValueBytes+1)}},
	} {
		if _, err := Parse(values, schema()); err == nil {
			t.Fatalf("query unexpectedly accepted: %#v", values)
		}
	}
	if _, err := Parse(url.Values{"sort": []string{"title;DROP TABLE users"}}, schema()); err == nil {
		t.Fatal("unsafe sort unexpectedly accepted")
	}
}

func TestParseSupportsBoundedOrGroup(t *testing.T) {
	values := url.Values{"where[or][0][status][equals]": []string{"draft"}, "where[or][1][status][equals]": []string{"published"}}
	parsed, err := Parse(values, schema())
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Conditions) != 2 || parsed.Conditions[0].Group != "or" || parsed.Conditions[1].Group != "or" {
		t.Fatalf("unexpected conditions: %#v", parsed.Conditions)
	}
	built, err := parsed.BuildSQL(database.SQLite, schema())
	if err != nil {
		t.Fatal(err)
	}
	if built.Where != "(status = ? OR status = ?)" {
		t.Fatalf("unexpected OR SQL: %s", built.Where)
	}
}

func TestParseAndBuildSQLRejectAmbiguousOrUnboundedInputs(t *testing.T) {
	parseFields := schema()
	parseFields["status"] = Field{Name: "status", Operators: map[string]bool{"exists": true, "in": true}}
	for name, values := range map[string]url.Values{
		"repeated scalar":       {"where[title][contains]": {"a", "b"}},
		"invalid exists":        {"where[status][exists]": {"perhaps"}},
		"page and offset":       {"page": {"2"}, "offset": {"10"}},
		"too many IN values":    {"where[status][in]": {strings.Repeat("x,", MaxInValues)}},
		"too many query keys":   makeQueryKeys(MaxQueryKeys + 1),
		"extra bracket segment": {"where[title][contains][ignored]": {"x"}},
		"invalid group index":   {"where[or][bad][title][equals]": {"x"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(values, parseFields); err == nil {
				t.Fatalf("accepted %#v", values)
			}
		})
	}
	fields := schema()
	fields["status"] = Field{Name: "status", Operators: map[string]bool{"exists": true, "in": true}}
	for name, q := range map[string]Query{
		"exists has strict bool": {Conditions: []Condition{{Field: "status", Operator: "exists", Values: []string{"no"}, Group: "and"}}, Limit: 25},
		"exists one value":       {Conditions: []Condition{{Field: "status", Operator: "exists", Values: []string{"true", "false"}, Group: "and"}}, Limit: 25},
		"scalar one value":       {Conditions: []Condition{{Field: "title", Operator: "contains", Values: []string{"a", "b"}, Group: "and"}}, Limit: 25},
		"identifier revalidated": {Conditions: []Condition{{Field: "title) OR 1=1 --", Operator: "contains", Values: []string{"x"}, Group: "and"}}, Limit: 25},
		"total args bounded":     {Conditions: manyInConditions(MaxTotalArgs + 1), Limit: 25},
		"pagination revalidated": {Limit: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := q.BuildSQL(database.SQLite, fields); err == nil {
				t.Fatalf("accepted %#v", q)
			}
		})
	}
}

func makeQueryKeys(n int) url.Values {
	values := make(url.Values, n)
	for i := 0; i < n; i++ {
		values["x"+strconv.Itoa(i)] = []string{"v"}
	}
	return values
}

func manyInConditions(n int) []Condition {
	conditions := make([]Condition, 0, n/32+1)
	for len(conditions)*MaxInValues < n {
		values := make([]string, MaxInValues)
		for i := range values {
			values[i] = "x"
		}
		conditions = append(conditions, Condition{Field: "status", Operator: "in", Values: values, Group: "and"})
	}
	return conditions
}
