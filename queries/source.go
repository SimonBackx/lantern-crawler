package queries

import (
	"index/suffixarray"
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
