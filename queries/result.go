package queries

import (
	"gopkg.in/mgo.v2/bson"
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
