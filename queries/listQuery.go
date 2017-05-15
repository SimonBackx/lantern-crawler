package queries

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/search"
)

type ListQuery struct {
	List     []string
	patterns []*search.Pattern
}

func (q *ListQuery) Execute(b []byte) [][]int {
	if q.patterns == nil {
		matcher := search.New(language.English, search.Loose)
		q.patterns = make([]*search.Pattern, len(q.List))
		for i, str := range q.List {
			q.patterns[i] = matcher.CompileString(str)
		}
	}

	for _, pattern := range q.patterns {
		start, end := pattern.Index(b)
		if start != -1 {
			return [][]int{[]int{start, end}}
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
