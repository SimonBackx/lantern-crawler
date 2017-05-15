package queries

import (
	"encoding/json"
	"fmt"
)

/// Kan een zoekoperatie uitvoeren op een stuk data.
/// Geeft true / false aan afhankelijk van de match
type QueryAction interface {
	Execute(s *Source) [][]int

	String() string
}

func UnmarshalQueryAction(b json.RawMessage, destination *QueryAction) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	var t string
	err = json.Unmarshal(*m["type"], &t)
	if err != nil {
		return err
	}

	switch t {
	case "regexp":
		var r RegexpQuery
		err := json.Unmarshal(b, &r)
		if err != nil {
			return err
		}
		*destination = &r
		return nil
	case "operator":
		var o OperatorQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	case "text":
		var o TextQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	case "list":
		var o ListQuery
		err := json.Unmarshal(b, &o)
		if err != nil {
			return err
		}
		*destination = &o
		return nil
	}

	return fmt.Errorf("Invalid type")
}
