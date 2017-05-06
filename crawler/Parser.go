package crawler

import (
	"bytes"
	"github.com/SimonBackx/lantern-crawler/queries"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
)

type ParseResult struct {
	Links   []*Link
	Results []*queries.Result
}

/*func byteArrayToString(b []byte) string {
    return strings.TrimSpace(string(b))
}*/

// Momenteel nog geen return value, dat is voor later
func Parse(reader io.Reader, queryList []queries.Query) (*ParseResult, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	htmlDoc, err := html.Parse(bytes.NewReader(data))
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

	FindLinks(htmlDoc, result)
	title := FindTitle(htmlDoc)

	byteString := NodeToText(htmlDoc)
	document := string(data[:])

	// Queries op uitvoeren
	for _, query := range queryList {
		snippet := query.Execute(byteString)
		if snippet != nil {
			apiResult := queries.NewResult(query, nil, nil, &document, title, snippet)
			result.Results = append(result.Results, apiResult)
		}
	}

	return result, nil
}

type Link struct {
	Href url.URL
}

func FindLinks(document *html.Node, result *ParseResult) {
	selector := cascadia.MustCompile("a")
	selection := selector.MatchAll(document)
	if selection == nil {
		return
	}

	links := make([]*Link, len(selection))

	errorOffset := 0
	for i, node := range selection {
		attr := NodeAttr(node, "href")
		if attr != nil {
			attrUrl, err := url.Parse(*attr)
			if err == nil {
				links[i-errorOffset] = &Link{*attrUrl}
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

	return
}

func FindTitle(document *html.Node) *string {
	selector := cascadia.MustCompile("head title")
	selection := selector.MatchFirst(document)
	if selection == nil {
		return nil
	}

	title := string(NodeToText(selection)[:])

	return queries.CleanString(&title)
}

func NodeAttr(node *html.Node, attrName string) *string {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return &attr.Val
		}
	}
	return nil
}

func NodeToText(node *html.Node) []byte {
	var buffer bytes.Buffer
	next := node.FirstChild
	depth := 0
	for next != nil && depth >= 0 {
		if next.Type == html.TextNode {
			buffer.WriteString(next.Data)
			buffer.WriteString("\n")
		}

		if next.FirstChild != nil && !(next.Type == html.ElementNode && (next.Data == "script" || next.Data == "style" || next.Data == "head" || next.Data == "noscript")) {
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
	return buffer.Bytes()
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
