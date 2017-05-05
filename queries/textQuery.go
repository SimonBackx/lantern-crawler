package queries

import (
	"encoding/json"
	"fmt"
)

type TextQuery struct {
	Text string
}

func (q *TextQuery) Execute(str *string) bool {
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
