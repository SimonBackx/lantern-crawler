package queries

import (
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"time"
)

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

func (q *Query) Execute(str *string) bool {
	return q.Query.Execute(str)
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
