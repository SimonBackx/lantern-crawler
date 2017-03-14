package parser

import (
	"bytes"
	"golang.org/x/net/html"
	"io"
	"regexp"
	"strings"
)

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
func Parse(reader io.Reader, parsers []IParser) (*ParseResult, error) {

	htmlDoc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	b := &buffer
	err2 := html.Render(b, htmlDoc)
	if err2 != nil {
		return nil, err2
	}
	result := &ParseResult{}

	for _, parser := range parsers {
		if !parser.MatchDocument(htmlDoc, result) {
			return nil, ParseError{"Error in parsing"}
		}
	}
	return result, nil
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
