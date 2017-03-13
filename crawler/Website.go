package crawler

import (
	"github.com/SimonBackx/master-project/parser"
	"net/url"
	"regexp"
)

type Website struct {
	Name                 string
	URL                  string
	ListingConfiguration *parser.ListingConfiguration
	ListingRegexp        *regexp.Regexp
	RunningRequests      int
	Paused               bool
}

func (web *Website) GetParsers(url *url.URL) []parser.IParser {
	if web.ListingRegexp.MatchString(url.EscapedPath()) {
		// Tis een listing :p
		parsers := make([]parser.IParser, 2, 2)
		parsers[0] = &parser.LinkParser{}
		parsers[1] = &parser.ListingParser{Configuration: web.ListingConfiguration}
		return parsers
	}

	parsers := make([]parser.IParser, 1, 1)
	parsers[0] = &parser.LinkParser{}
	return parsers
}
