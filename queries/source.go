package queries

import (
	"index/suffixarray"
	"unicode"
)

type Source struct {
	Text  []byte
	Index *suffixarray.Index
}

func NewSource(text []byte) *Source {
	return &Source{Text: text}
}

func (s *Source) GetOrCreateIndex() *suffixarray.Index {
	if s.Index == nil {
		s.Index = suffixarray.New(s.Text)
	}
	return s.Index
}

func (s *Source) Lookup(needle []byte) []int {
	index := s.GetOrCreateIndex()
	start := index.Lookup([]byte(needle), 0)

	if start != nil {
		// Staat het woord wel apart?
		for index := range start {
			endPosition := start[0] + len(needle)

			endOk := false
			startOk := false

			// Voorwaarde: woord staat apart
			if endPosition >= len(s.Text) {
				endOk = true
			} else if unicode.IsSpace(rune(s.Text[endPosition])) {
				endOk = true
			}

			if index <= 0 {
				startOk = true
			} else if unicode.IsSpace(rune(s.Text[index-1])) {
				startOk = true
			}

			if startOk && endOk {
				return []int{index, endPosition}
			}
		}
	}
	return nil
}
