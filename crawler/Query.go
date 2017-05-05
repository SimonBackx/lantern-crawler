// TODO: create seperate package for data objects

package crawler

import (
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"regexp"
	"time"
)

type Result struct {
	Id      bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	QueryId bson.ObjectId `json:"queryId" bson:"queryId"`

	LastFound   time.Time `json:"lastFound" bson:"lastFound"`
	CreatedOn   time.Time `json:"createdOn" bson:"createdOn"`
	Occurrences int       `json:"occurrences" bson:"occurrences"`
	Url         string    `json:"url" bson:"url"`
	Body        *string   `json:"body" bson:"body"`
}

func NewResult(query Query, url string, body *string) *Result {
	return &Result{QueryId: query.Id, LastFound: time.Now(), CreatedOn: time.Now(), Occurrences: 1, Url: url, Body: body}
}

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

/// Kan een zoekoperatie uitvoeren op een stuk data.
/// Geeft true / false aan afhankelijk van de match
type QueryAction interface {
	query(str *string) bool
	String() string
}

func UnmarshalQueryAction(b json.RawMessage, destination *QueryAction) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	var t string
	err = json.Unmarshal(*m["type"], &t)
	if err != nil {
		return err
	}

	switch t {
	case "regexp":
		var r RegexpQuery
		err := json.Unmarshal(b, &r)
		if err != nil {
			return err
		}
		*destination = &r
		return nil
	case "operator":
		var o OperatorQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	case "text":
		var o TextQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	case "list":
		var o ListQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	}

	return fmt.Errorf("Invalid type")
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

func (o *OperatorQuery) query(str *string) bool {
	first := o.First.query(str)
	if first {
		if o.Operator == OrOperator {
			return true
		}
	} else {
		if o.Operator == AndOperator {
			return false
		}
	}
	return o.Last.query(str)
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

/// All supported basic actions
type RegexpQuery struct {
	Regexp *regexp.Regexp
}

func (o *RegexpQuery) MarshalJSON() ([]byte, error) {
	m := make(map[string]string)
	m["regexp"] = o.Regexp.String()
	m["type"] = "regexp"
	return json.Marshal(m)
}

func (o *RegexpQuery) UnmarshalJSON(b []byte) error {
	// First, deserialize everything into a map of map
	var objMap map[string]string
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	if len(objMap["regexp"]) < 1 {
		return fmt.Errorf("Empty regexp not allowed")
	}

	o.Regexp, err = regexp.Compile(objMap["regexp"])
	if err != nil {
		return err
	}
	return nil
}

func NewRegexpQuery(str string) *RegexpQuery {
	reg := regexp.MustCompile(str)
	return &RegexpQuery{Regexp: reg}
}

func (a *RegexpQuery) query(str *string) bool {
	return a.Regexp.MatchString(*str)
}

func (q *RegexpQuery) String() string {
	return q.Regexp.String()
}

type ListQuery struct {
	List []string
}

func (q *ListQuery) query(str *string) bool {
	// todo: not implemented
	return false
}

func (q *ListQuery) String() string {
	return fmt.Sprintf("List[%v]", len(q.List))
}

func (q *ListQuery) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["list"] = q.List
	m["type"] = "list"
	return json.Marshal(m)
}

type TextQuery struct {
	Text string
}

func (q *TextQuery) query(str *string) bool {
	// todo: not implemented
	return false
}

func (q *TextQuery) String() string {
	return fmt.Sprintf("\"%s\"", q.Text)
}

func (q *TextQuery) MarshalJSON() ([]byte, error) {
	m := make(map[string]string)
	m["text"] = q.Text
	m["type"] = "text"
	return json.Marshal(m)
}

type Query struct {
	Id        bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Name      string        `json:"name" bson:"name"`
	CreatedOn time.Time     `json:"createdOn" bson:"createdOn"`
	Query     QueryAction   `json:"root" bson:"root"`
}

func NewQuery(name string, q QueryAction) *Query {
	now := time.Now()
	return &Query{Name: name, CreatedOn: now, Query: q}
}

/*func (q *Query) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["_id"] = q.Id
	m["name"] = q.Name
	m["createdOn"] = q.CreatedOn
	m["root"] = q.Query
	return json.Marshal(m)
}

func (q *Query) MarshalJSONWithoutId() ([]byte, error) {
	m := make(map[string]interface{})
	m["name"] = q.Name
	m["createdOn"] = q.CreatedOn
	m["root"] = q.Query
	return json.Marshal(m)
}*/

func (q *Query) UnmarshalJSON(b []byte) error {

	// First, deserialize everything into a map of map
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	if objMap["name"] == nil || objMap["root"] == nil {
		return fmt.Errorf("name and/or root not set")
	}

	err = json.Unmarshal(*objMap["name"], &q.Name)
	if err != nil {
		return err
	}

	if objMap["_id"] != nil {
		var id string
		err = json.Unmarshal(*objMap["_id"], &id)
		if err != nil {
			return err
		}
		q.Id = bson.ObjectIdHex(id)
	}

	if objMap["createdOn"] != nil {
		err = json.Unmarshal(*objMap["createdOn"], &q.CreatedOn)
		if err != nil {
			return err
		}
	} else {
		q.CreatedOn = time.Now()
	}

	err = UnmarshalQueryAction(*objMap["root"], &q.Query)
	if err != nil {
		return err
	}
	return nil
}

func (q *Query) String() string {
	return q.Query.String()
}

func (q *Query) JSON() ([]byte, error) {
	return json.Marshal(q)
}
