package queries

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/search"
)

type TextQuery struct {
	Text    string
	pattern *search.Pattern
}

func (q *TextQuery) Execute(b []byte) [][]int {
	if q.pattern == nil {
		matcher := search.New(language.English, search.Loose)
		q.pattern = matcher.CompileString(q.Text)
	}

	start, end := q.pattern.Index(b)
	if start != -1 {
		return [][]int{[]int{start, end}}
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
