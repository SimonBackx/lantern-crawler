package parser

import (
	"fmt"
	"golang.org/x/net/html"
)

func PrintLinks(links []*Link) {
	for _, link := range links {
		fmt.Println(link.Anchor, " > ", link.Href.String())
	}
}

type IParser interface {
	MatchDocument(document *html.Node, result *ParseResult) bool
}
