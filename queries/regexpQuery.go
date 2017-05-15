package queries

import (
	"encoding/json"
	"fmt"
	"regexp"
)

/// All supported basic actions
type RegexpQuery struct {
	Regexp *regexp.Regexp
}

func (o *RegexpQuery) MarshalJSON() ([]byte, error) {
	m := make(map[string]string)
	m["regexp"] = o.Regexp.String()
	m["type"] = "regexp"
	return json.Marshal(m)
}

func (o *RegexpQuery) UnmarshalJSON(b []byte) error {
	// First, deserialize everything into a map of map
	var objMap map[string]string
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	if len(objMap["regexp"]) < 1 {
		return fmt.Errorf("Empty regexp not allowed")
	}

	o.Regexp, err = regexp.Compile(objMap["regexp"])
	if err != nil {
		return err
	}
	return nil
}

func NewRegexpQuery(str string) *RegexpQuery {
	reg := regexp.MustCompile(str)
	return &RegexpQuery{Regexp: reg}
}

func (a *RegexpQuery) Execute(s *Source) [][]int {
	index := s.GetOrCreateIndex()
	return index.FindAllIndex(a.Regexp, 1)
}

func (q *RegexpQuery) String() string {
	return q.Regexp.String()
}
