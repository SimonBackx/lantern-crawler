package parser

import (
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"net/url"
)

type Link struct {
	Anchor string
	Href   url.URL
}

type LinkParser struct {
}

func (parser *LinkParser) MatchDocument(document *html.Node, result *ParseResult) bool {
	selector := cascadia.MustCompile("a")
	selection := selector.MatchAll(document)
	if selection == nil {
		return true
	}

	links := make([]*Link, len(selection))
	for i, node := range selection {
		attr := NodeAttr(node, "href")
		if attr != nil {
			attrUrl, err := url.Parse(*attr)
			if err == nil {
				links[i] = &Link{NodeToText(node), *attrUrl}
			}
		}
	}

	result.Links = links

	return true
}
