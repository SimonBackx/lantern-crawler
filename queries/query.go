package queries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"strings"
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

func (q *Query) Execute(b []byte) *string {
	result := q.Query.Execute(b)
	if result == nil || len(result) == 0 {
		return nil
	}

	var maxLength int = 120
	characters := maxLength / len(result)

	enddot := true
	var buffer bytes.Buffer
	for i, resultIndexes := range result {
		if i > 0 {
			buffer.WriteString("... ")
		}
		enddot = true

		start := resultIndexes[0]
		end := resultIndexes[1]
		length := end - start
		margin := (characters - length) / 2

		if margin > 0 {
			foundStart := start
			prev := false
			predot := (i == 0)
			for i := start; i >= start-margin; i-- {
				if i <= 0 {
					foundStart = 0
					predot = false
					break
				}

				if b[i] == " "[0] {
					if !prev {
						foundStart = i + 1
						prev = true
					}
				} else if b[i] == "\n"[0] {
					foundStart = i + 1
					predot = false
					break
				} else {
					prev = false
				}
			}

			start = foundStart

			foundEnd := end
			prev = false
			for i := end; i <= end+margin; i++ {
				if i >= len(b) {
					foundEnd = len(b)
					enddot = false
					break
				}

				if b[i] == " "[0] {
					if !prev {
						foundEnd = i
						prev = true
						enddot = true
					}
				} else if b[i] == "\n"[0] {
					foundEnd = i
					enddot = false
					break
				} else {
					prev = false
				}
			}

			end = foundEnd

			if predot {
				buffer.WriteString("...")
			}

		} else {
			end = start + characters
			if end > len(b) {
				end = len(b)
			}
		}

		buffer.Write(b[start:end])

	}

	if enddot {
		buffer.WriteString("... ")
	}
	str := buffer.String()
	str = strings.Replace(str, "\n", "", -1)
	return &str
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
