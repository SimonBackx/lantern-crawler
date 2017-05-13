package queries

import (
	"encoding/json"
	"fmt"
)

type Operator string

// OperatorQuerys
const (
	AndOperator Operator = "AND"
	OrOperator           = "OR"
)

func (op Operator) String() string {
	if op == AndOperator {
		return "AND"
	}
	return "OR"
}

/// Een QueryAction die bestaat uit 2 actions met een operator zoals AND of OR
/// ertussen. Bij and en eerste is false geeft ze meteen false terug
/// Bij OR en eerste is true, geeft het meteen true terug.
type OperatorQuery struct {
	First    QueryAction
	Operator Operator
	Last     QueryAction
}

func (o *OperatorQuery) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	m["first"] = o.First
	m["last"] = o.Last
	m["operator"] = o.Operator
	m["type"] = "operator"
	return json.Marshal(m)
}

func (o *OperatorQuery) Execute(b []byte) [][]int {
	first := o.First.Execute(b)
	if first != nil {
		if o.Operator == OrOperator {
			return first
		}
	} else {
		if o.Operator == AndOperator {
			return nil
		}
	}

	last := o.Last.Execute(b)
	if last == nil {
		return nil
	}

	if first == nil {
		return last
	}

	return append(first, last...)
}

func (o *OperatorQuery) UnmarshalJSON(b []byte) error {
	// First, deserialize everything into a map of map
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	if objMap["operator"] == nil || objMap["first"] == nil || objMap["last"] == nil {
		return fmt.Errorf("Json: OperatorQuery's operator, first and/or last not set")
	}

	err = json.Unmarshal(*objMap["operator"], &o.Operator)
	if err != nil {
		return err
	}

	if o.Operator != "AND" && o.Operator != "OR" {
		return fmt.Errorf("Json: OperatorQuery invalid operator '%v'", o.Operator)
	}

	err = UnmarshalQueryAction(*objMap["first"], &o.First)
	if err != nil {
		return err
	}

	err = UnmarshalQueryAction(*objMap["last"], &o.Last)
	if err != nil {
		return err
	}
	return nil
}

func (o *OperatorQuery) String() string {
	return "(" + o.First.String() + " " + o.Operator.String() + " " + o.Last.String() + ")"
}

func NewOperatorQuery(first QueryAction, operator Operator, last QueryAction) *OperatorQuery {
	return &OperatorQuery{First: first, Operator: operator, Last: last}
}
