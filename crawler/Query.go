package crawler

import (
	"regexp"
)

type QueryOperator int

// QueryOperators
const (
	AndOperator QueryOperator = iota
	OrOperator
)

func (op QueryOperator) String() string {
	if op == AndOperator {
		return "AND"
	}
	return "OR"
}

/// Kan een zoekoperatie uitvoeren op een stuk data.
/// Geeft true / false aan afhankelijk van de match
type QueryAction interface {
	query(str *string) bool
	String() string
}

/// Een QueryAction die bestaat uit 2 actions met een operator zoals AND of OR
/// ertussen. Bij and en eerste is false geeft ze meteen false terug
/// Bij OR en eerste is true, geeft het meteen true terug.
type QueryOperation struct {
	First     QueryAction
	Operation QueryOperator
	Last      QueryAction
}

func (o *QueryOperation) query(str *string) bool {
	first := o.First.query(str)
	if first {
		if o.Operation == OrOperator {
			return true
		}
	} else {
		if o.Operation == AndOperator {
			return false
		}
	}
	return o.Last.query(str)
}

func (o *QueryOperation) String() string {
	return "(" + o.First.String() + " " + o.Operation.String() + " " + o.Last.String() + ")"
}

func NewQueryOperation(first QueryAction, operation QueryOperator, last QueryAction) *QueryOperation {
	return &QueryOperation{First: first, Operation: operation, Last: last}
}

/// All supported basic actions
type QueryRegexp struct {
	Regexp *regexp.Regexp
}

func NewQueryRegexp(str string) *QueryRegexp {
	reg := regexp.MustCompile(str)
	return &QueryRegexp{Regexp: reg}
}

func (a *QueryRegexp) query(str *string) bool {
	return a.Regexp.MatchString(*str)
}

func (q *QueryRegexp) String() string {
	return q.Regexp.String()
}

type QueryList struct {
}
type QueryText struct {
}

type Query struct {
	Query QueryAction
}

func NewQuery(q QueryAction) *Query {
	return &Query{Query: q}
}

func (q *Query) String() string {
	return q.Query.String()
}
