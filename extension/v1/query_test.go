package v1

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestParseResourceQueryBoundsAndAllowlist(t *testing.T) {
	fields := map[string]QueryField{"title": {Name: "title", Operators: map[string]bool{"contains": true}}}
	q, err := ParseResourceQuery(url.Values{"where[title][contains]": {"roof"}, "limit": {"10"}, "sort": {"-title"}}, fields)
	if err != nil {
		t.Fatal(err)
	}
	if q.Limit != 10 || len(q.Conditions) != 1 || !q.Sort[0].Desc {
		t.Fatalf("unexpected query: %#v", q)
	}
	if _, err := ParseResourceQuery(url.Values{"where[title][equals]": {"roof"}}, fields); err == nil {
		t.Fatal("expected operator rejection")
	}
}

func TestResourceQueryBuildSQLBindsValues(t *testing.T) {
	fields := map[string]QueryField{"title": {Name: "title", Operators: map[string]bool{"contains": true}}}
	q, err := ParseResourceQuery(url.Values{"where[title][contains]": {"x' OR 1=1"}, "sort": {"-title"}}, fields)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := q.BuildSQL("postgres", fields)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Where != "title LIKE $1" || len(sql.Args) != 1 || sql.Order != "title DESC" {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
}

func TestResourceQueryBuildSQLGroupsOrAndRange(t *testing.T) {
	fields := map[string]QueryField{
		"title":    {Name: "title"},
		"priority": {Name: "priority"},
	}
	values := url.Values{
		"where[or][0][title][equals]":      {"one"},
		"where[or][1][title][equals]":      {"two"},
		"where[priority][less_than_equal]": {"3"},
	}
	q, err := ParseResourceQuery(values, fields)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := q.BuildSQL("sqlite", fields)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Where != "priority <= ? AND (title = ? OR title = ?)" {
		t.Fatalf("unexpected grouped SQL: %q", sql.Where)
	}
	if len(sql.Args) != 3 {
		t.Fatalf("args = %#v", sql.Args)
	}
}

func TestResourceQueryRejectsAmbiguousAndDirectUnboundedAST(t *testing.T) {
	fields := map[string]QueryField{
		"title":  {Name: "title", Operators: map[string]bool{"contains": true}},
		"status": {Name: "status", Operators: map[string]bool{"exists": true, "in": true}},
	}
	for name, values := range map[string]url.Values{
		"repeated scalar": {"where[title][contains]": {"a", "b"}},
		"invalid exists":  {"where[status][exists]": {"yes"}},
		"page and offset": {"page": {"2"}, "offset": {"10"}},
		"too many IN":     {"where[status][in]": {strings.Repeat("x,", MaxQueryInValues)}},
		"too many keys":   makePublicQueryKeys(MaxQueryKeys + 1),
		"extra segment":   {"where[title][contains][ignored]": {"x"}},
		"invalid index":   {"where[or][bad][title][equals]": {"x"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResourceQuery(values, fields); err == nil {
				t.Fatalf("accepted %#v", values)
			}
		})
	}
	for name, q := range map[string]ResourceQuery{
		"strict exists":      {Conditions: []QueryCondition{{Field: "status", Operator: "exists", Values: []string{"1"}, Group: "and"}}, Limit: 25},
		"scalar cardinality": {Conditions: []QueryCondition{{Field: "title", Operator: "contains", Values: []string{"a", "b"}, Group: "and"}}, Limit: 25},
		"group allowlist":    {Conditions: []QueryCondition{{Field: "title", Operator: "contains", Values: []string{"x"}, Group: "anything"}}, Limit: 25},
		"total args":         {Conditions: publicManyInConditions(MaxQueryTotalArgs + 1), Limit: 25},
		"limit required":     {Limit: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := q.BuildSQL("sqlite", fields); err == nil {
				t.Fatalf("accepted %#v", q)
			}
		})
	}
}

func makePublicQueryKeys(n int) url.Values {
	values := make(url.Values, n)
	for i := 0; i < n; i++ {
		values["x"+strconv.Itoa(i)] = []string{"v"}
	}
	return values
}

func publicManyInConditions(n int) []QueryCondition {
	conditions := make([]QueryCondition, 0, n/MaxQueryInValues+1)
	for len(conditions)*MaxQueryInValues < n {
		values := make([]string, MaxQueryInValues)
		for i := range values {
			values[i] = "x"
		}
		conditions = append(conditions, QueryCondition{Field: "status", Operator: "in", Values: values, Group: "and"})
	}
	return conditions
}
