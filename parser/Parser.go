package parser

import (
	"bytes"
	"github.com/SimonBackx/lantern-crawler/crawler"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"io"
	"net/url"
	"regexp"
	"strings"
)

type ParseResult struct {
	Success bool
	Retry   bool // Opnieuw proberen met opgegeven ErrorParser
	Listing *Listing
	Links   []*Link
	Queries []*crawler.Query
}

func byteArrayToString(b []byte) string {
	return strings.TrimSpace(string(b))
}

type ParseError struct {
	msg string
}

func (e ParseError) Error() string {
	return e.msg
}

// Momenteel nog geen return value, dat is voor later
func Parse(reader io.Reader, parsers []IParser, queries []*crawler.Query) (*ParseResult, error) {

	htmlDoc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	/*var buffer bytes.Buffer
	b := &buffer
	err2 := html.Render(b, htmlDoc)
	if err2 != nil {
		return nil, err2
	}*/
	result := &ParseResult{}

	for _, parser := range parsers {
		if !parser.MatchDocument(htmlDoc, result) {
			return nil, ParseError{"Error in parsing"}
		}
	}

	str := NodeToText(htmlDoc)

	// Queries op uitvoeren
	for _, query := range queries {
		matched := query.Query.query(&str)
		if matched {
			result.Queries = append(result.Queries, query)
		}
	}

	return result, nil
}

func FindLinks(document *html.Node, result *ParseResult) bool {
	selector := cascadia.MustCompile("a")
	selection := selector.MatchAll(document)
	if selection == nil {
		return true
	}

	links := make([]*Link, len(selection))

	errorOffset := 0
	for i, node := range selection {
		attr := NodeAttr(node, "href")
		if attr != nil {
			attrUrl, err := url.Parse(*attr)
			if err == nil {
				links[i-errorOffset] = &Link{NodeToText(node), *attrUrl}
			} else {
				errorOffset++
			}
		} else {
			errorOffset++
		}
	}

	if errorOffset != 0 {
		links = links[0 : len(links)-errorOffset]
	}

	result.Links = links

	return true
}

func NodeAttr(node *html.Node, attrName string) *string {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return &attr.Val
		}
	}
	return nil
}

func NodeToText(node *html.Node) string {
	var buffer bytes.Buffer
	next := node.FirstChild
	depth := 0
	for next != nil && depth >= 0 {
		if next.Type == html.TextNode {
			buffer.WriteString(next.Data)
		}

		if next.FirstChild != nil {
			next = next.FirstChild
			depth++
		} else {
			if next.NextSibling == nil {
				next = next.Parent.NextSibling
				depth--
			} else {
				next = next.NextSibling
			}
		}
	}
	return buffer.String()
}

func cleanString(str string) string {
	beginRegex := regexp.MustCompile("((^[^\\S\n\r]+)|([^\\S\n\r]+$))")
	spaceRegex := regexp.MustCompile("[^\\S\n\r]+")

	// Lange sequenties eruit halen
	weirdWords := regexp.MustCompile("[^\\s]{60,}|[-_=+*]{5,}")
	return weirdWords.ReplaceAllString(spaceRegex.ReplaceAllString(beginRegex.ReplaceAllString(str, ""), " "), "")
}

func shortDescription(str string) string {
	reg := regexp.MustCompile("[\n\r]+")
	str = reg.ReplaceAllString(str, " ")
	if len(str) > 120 {
		s := []string{str[0:120], "..."}
		return strings.Join(s, "")
	}
	return str
}
