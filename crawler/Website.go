package crawler

import (
	"github.com/SimonBackx/master-project/parser"
	"net/url"
	"regexp"
)

type Website struct {
	Name            string
	URL             string
	RunningRequests int
	MaxRequests     int

	ListingConfiguration *parser.ListingConfiguration // Kan nil zijn
	ListingRegexp        *regexp.Regexp               // Kan nil zijn
}

func (web *Website) GetParsers(url *url.URL) []parser.IParser {
	if url == nil {
		panic("Url is nil")
	}

	if web == nil {
		panic("Website is nil")
	}

	if web.ListingRegexp != nil && web.ListingRegexp.MatchString(url.EscapedPath()) {
		// Tis een listing :p
		parsers := make([]parser.IParser, 2, 2)
		parsers[0] = &parser.LinkParser{}
		parsers[1] = &parser.ListingParser{Configuration: web.ListingConfiguration}
		return parsers
	}

	// Only search for links
	parsers := make([]parser.IParser, 1, 1)
	parsers[0] = &parser.LinkParser{}
	return parsers
}

var websiteMap map[string]*Website

func InitialiseWebsites() {
	websiteMap = make(map[string]*Website, 0)
}

/**
 * Creates website if it doesn't exists
 */
func GetWebsiteForDomain(domain string) *Website {
	web := websiteMap[domain]
	if web == nil {
		web = &Website{URL: domain}
		AddWebsite(web)
	}

	return web
}

func AddWebsite(web *Website) {
	websiteMap[web.URL] = web
}
