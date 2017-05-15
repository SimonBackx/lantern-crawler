package queries

import (
	"encoding/json"
	"fmt"
)

type ListQuery struct {
	List []string
}

func (q *ListQuery) Execute(s *Source) [][]int {
	index := s.GetOrCreateIndex()

	for _, str := range q.List {
		start := index.Lookup([]byte(str), 1)

		if start != nil {
			return [][]int{[]int{start[0], start[0] + len(str)}}
		}
	}

	return nil
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
