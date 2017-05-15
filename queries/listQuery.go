package queries

import (
	"encoding/json"
	"fmt"
)

type ListQuery struct {
	List []string
}

func (q *ListQuery) Execute(s *Source) [][]int {
	for _, str := range q.List {
		position := s.Lookup([]byte(str))
		if position == nil {
			continue
		}

		return [][]int{position}
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
