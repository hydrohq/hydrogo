package query

import (
	"net/url"
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
