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
	start := index.Lookup([]byte(needle), -1)

	if start != nil {
		// Staat het woord wel apart?
		for _, index := range start {
			endPosition := index + len(needle)

			endOk := false
			startOk := false

			// Voorwaarde: woord staat apart
			if endPosition >= len(s.Text) {
				endOk = true
			} else if s.Text[endPosition] == '0' || unicode.IsSpace(rune(s.Text[endPosition])) {
				endOk = true
			}

			if index <= 0 {
				startOk = true
			} else if s.Text[index-1] == '0' || unicode.IsSpace(rune(s.Text[index-1])) {
				startOk = true
			}

			if startOk && endOk {
				return []int{index, endPosition}
			}
		}
	}
	return nil
}
