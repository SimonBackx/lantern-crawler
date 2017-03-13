package parser

import (
	"golang.org/x/net/html"
)

type Link struct {
}

type LinkParser struct {
	Links []Link
}

func (parser *LinkParser) MatchDocument(document *html.Node, result *ParseResult) bool {
	// todo
	return true
}
