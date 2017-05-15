package queries

import (
	"encoding/json"
	"fmt"
)

type TextQuery struct {
	Text string
}

func (q *TextQuery) Execute(s *Source) [][]int {
	position := s.Lookup([]byte(q.Text))
	if position == nil {
		return nil
	}

	return [][]int{position}
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
