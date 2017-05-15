package queries

import (
	"encoding/json"
	"fmt"
)

type TextQuery struct {
	Text string
}

func (q *TextQuery) Execute(s *Source) [][]int {
	index := s.GetOrCreateIndex()

	start := index.Lookup([]byte(q.Text), 1)

	if start != nil {
		return [][]int{[]int{start[0], start[0] + len(q.Text)}}
	}

	return nil
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
