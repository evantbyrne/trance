package trance

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestFilterAnd(t *testing.T) {
	clauses := And(
		"SKIP",
		FilterClause{Rule: "A"},
		FilterClause{Rule: "B"},
		And(
			FilterClause{Rule: "C.1"},
			FilterClause{Rule: "C.2"},
		),
	)

	expected := []FilterClause{
		{Rule: "("},
		{Rule: "A"},
		{Rule: "AND"},
		{Rule: "B"},
		{Rule: "AND"},
		{Rule: "("},
		{Rule: "C.1"},
		{Rule: "AND"},
		{Rule: "C.2"},
		{Rule: ")"},
		{Rule: ")"},
	}
	if !slices.Equal(clauses, expected) {
		t.Errorf("Expected '%+v', got '%+v'", expected, clauses)
	}
}

func TestFilterNot(t *testing.T) {
	clauses := And(
		"SKIP",
		FilterClause{Rule: "A"},
		Not("B", "=", "foo"),
		Or(
			Not("C.1", "=", "bar"),
			FilterClause{Rule: "C.2"},
		),
	)

	expected := []FilterClause{
		{Rule: "("},
		{Rule: "A"},
		{Rule: "AND"},
		{Rule: "NOT"},
		{Rule: "("},
		{Left: "B", Operator: "=", Right: "foo", Rule: "WHERE"},
		{Rule: ")"},
		{Rule: "AND"},
		{Rule: "("},
		{Rule: "NOT"},
		{Rule: "("},
		{Left: "C.1", Operator: "=", Right: "bar", Rule: "WHERE"},
		{Rule: ")"},
		{Rule: "OR"},
		{Rule: "C.2"},
		{Rule: ")"},
		{Rule: ")"},
	}
	if !slices.Equal(clauses, expected) {
		t.Errorf("Expected '%+v', got '%+v'", expected, clauses)
	}
}

func TestFilterOr(t *testing.T) {
	clauses := Or(
		FilterClause{Rule: "A"},
		"SKIP",
		Or(
			FilterClause{Rule: "B.1"},
			FilterClause{Rule: "B.2"},
		),
		FilterClause{Rule: "C"},
	)

	expected := []FilterClause{
		{Rule: "("},
		{Rule: "A"},
		{Rule: "OR"},
		{Rule: "("},
		{Rule: "B.1"},
		{Rule: "OR"},
		{Rule: "B.2"},
		{Rule: ")"},
		{Rule: "OR"},
		{Rule: "C"},
		{Rule: ")"},
	}
	if !slices.Equal(clauses, expected) {
		t.Errorf("Expected '%+v', got '%+v'", expected, clauses)
	}
}

func TestFlattenFilterClause(t *testing.T) {
	clauses := []any{
		FilterClause{Rule: "A"},
		FilterClause{Rule: "B"},
		[]FilterClause{
			{Rule: "C.1"},
			{Rule: "C.2"},
		},
		"SKIP",
		FilterClause{Rule: "D"},
	}
	expected := []FilterClause{
		{Rule: "Z"},
		{Rule: "A"},
		{Rule: "B"},
		{Rule: "C.1"},
		{Rule: "C.2"},
		{Rule: "D"},
	}
	flat := []FilterClause{
		{Rule: "Z"},
	}
	for _, clause := range clauses {
		flat = flattenFilterClause(flat, clause)
	}
	if !slices.Equal(flat, expected) {
		t.Errorf("Expected '%+v', got '%+v'", expected, flat)
	}
}
