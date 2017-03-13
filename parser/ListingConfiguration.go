package parser

import (
	"github.com/andybalholm/cascadia"
)

type ListingConfiguration struct {
	TitleSelector       cascadia.Selector
	DescriptionSelector cascadia.Selector
	AuthorSelector      cascadia.Selector
	PriceSelector       cascadia.Selector
}

func NewListingConfiguration(title, description, author, price string) *ListingConfiguration {
	return &ListingConfiguration{
		TitleSelector:       cascadia.MustCompile(title),
		DescriptionSelector: cascadia.MustCompile(description),
		AuthorSelector:      cascadia.MustCompile(author),
		PriceSelector:       cascadia.MustCompile(price),
	}
}
