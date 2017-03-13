package parser

import (
	"golang.org/x/net/html"
)

type ParseResult struct {
	Success bool
	Retry   bool // Opnieuw proberen met opgegeven ErrorParser
	Listing *Listing
	Links   []Link
}

type IParser interface {
	MatchDocument(document *html.Node, result *ParseResult) bool
}
