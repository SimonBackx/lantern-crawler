package parser

import (
	"fmt"
	"golang.org/x/net/html"
)

type ParseResult struct {
	Success bool
	Retry   bool // Opnieuw proberen met opgegeven ErrorParser
	Listing *Listing
	Links   []*Link
}

func PrintLinks(links []*Link) {
	for _, link := range links {
		fmt.Println(link.Anchor, " > ", link.Href)
	}
}

type IParser interface {
	MatchDocument(document *html.Node, result *ParseResult) bool
}
