package crawler

import (
	"bytes"
	"github.com/SimonBackx/lantern-crawler/queries"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
)

type ParseResult struct {
	Urls    []*url.URL
	Results []*queries.Result
	Title   *string
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
	document := string(data[:])
	result := ReadHtml(data)
	// Vanaf nu zijn alle tags etc uit data gefilterd en door lege characters
	// vervangen

	// Queries op uitvoeren
	for _, query := range queryList {
		snippet := query.Execute(data)
		if snippet != nil {
			apiResult := queries.NewResult(query, nil, nil, &document, result.Title, snippet)
			result.Results = append(result.Results, apiResult)
		}
	}

	return result, nil
}

func ReadHtml(data []byte) *ParseResult {
	reader := NewPositionReader(bytes.NewReader(data))

	head_depth := 0
	title_depth := 0
	var title string
	result := &ParseResult{Urls: make([]*url.URL, 0)}

	z := html.NewTokenizer(reader)

	previousEnd := 0
	ignore_depth := 0

	for {
		tt := z.Next()
		// Bytes worden hergebruikt, dus als ze
		// opgeslagen moeten worden is een kopie noodzakelijk
		switch tt {
		case html.ErrorToken:
			// Al de rest nog wissen
			data = data[:previousEnd]
			return result

		case html.TextToken:
			if ignore_depth == 0 && head_depth == 0 {
				endIndex := reader.Position - len(z.Buffered())
				str := z.Raw()
				startIndex := endIndex - len(str)

				// Nu alles verwijderen van previousEnd tot startIndex
				for i := previousEnd; i < startIndex; i++ {
					data[i] = 0
				}
				previousEnd = endIndex
			}

			if title_depth == 1 {
				// kopie maken
				title = string(z.Text())
				result.Title = &title
			}

		// Links detecteren
		case html.StartTagToken:
			tn, _ := z.TagName()

			if len(tn) == 1 && tn[0] == 'a' {
				key, val, moreAttr := z.TagAttr()
				for key != nil {

					if string(key) == "href" {
						attrUrl, _ := ParseUrlFromHref(val)
						if attrUrl != nil {
							result.Urls = append(result.Urls, attrUrl)
						}
						break
					}

					if !moreAttr {
						break
					}
					key, val, moreAttr = z.TagAttr()
				}

			} else if string(tn) == "head" {
				head_depth++
			} else if head_depth > 0 && string(tn) == "title" {
				title_depth++
			} else if string(tn) == "script" || string(tn) == "noscript" || string(tn) == "style" {
				ignore_depth++
			}

		case html.EndTagToken:
			tn, _ := z.TagName()
			if string(tn) == "head" {
				head_depth--
			} else if head_depth > 0 && string(tn) == "title" {
				title_depth--
			} else if string(tn) == "script" || string(tn) == "noscript" || string(tn) == "style" {
				ignore_depth--
			}

		}
		// Process the current token.
	}

	// nooit gebruikt
	return result
}

func ParseUrlFromHref(href []byte) (*url.URL, error) {
	startIndex := 0
	endIndex := len(href) - 1

	for startIndex <= endIndex && (href[startIndex] == ' ' || href[startIndex] == '\t' || href[startIndex] == '\n' || href[startIndex] == '\f' || href[startIndex] == '\r') {
		startIndex++
	}

	for endIndex >= 0 && (href[endIndex] == ' ' || href[endIndex] == '\t' || href[endIndex] == '\n' || href[endIndex] == '\f' || href[endIndex] == '\r') {
		endIndex--
	}

	if startIndex >= endIndex+1 || startIndex >= len(href) || endIndex < 0 {
		return nil, nil
	}

	u, err := url.Parse(string(href[startIndex : endIndex+1]))
	if err != nil {
		return u, err
	}
	u.Fragment = ""
	return u, err
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
