package parser

import (
	"fmt"
	"golang.org/x/net/html"
)

type ListingParser struct {
	Title       *string
	Description *string
	Author      *string
	Price       *string

	Configuration *ListingConfiguration
}

func (parser *ListingParser) MatchDocument(document *html.Node, result *ParseResult) bool {
	selection := parser.Configuration.TitleSelector.MatchFirst(document)
	if selection == nil {
		fmt.Println("Title not found")
		return false
	}
	title := cleanString(NodeToText(selection))

	selection = parser.Configuration.DescriptionSelector.MatchFirst(document)
	if selection == nil {
		fmt.Println("Description not found")
		return false
	}
	description := cleanString(NodeToText(selection))

	selection = parser.Configuration.AuthorSelector.MatchFirst(document)
	if selection == nil {
		fmt.Println("Author not found")
		return false
	}
	author := cleanString(NodeToText(selection))

	selection = parser.Configuration.PriceSelector.MatchFirst(document)
	if selection == nil {
		fmt.Println("Author not found")
		return false
	}
	price := cleanString(NodeToText(selection))

	listing := &Listing{Title: title, Description: description, Author: author, Price: price}
	result.Listing = listing

	return true
}
