package crawler

import (
	"bytes"
	"fmt"
	"github.com/SimonBackx/lantern-crawler/queries"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/url"
)

type ParseResult struct {
	Urls       []*url.URL
	Results    []*queries.Result
	Title      *string
	Lowercased []byte
}

/*func byteArrayToString(b []byte) string {
    return strings.TrimSpace(string(b))
}*/

// Momenteel nog geen return value, dat is voor later
func Parse(reader io.Reader, queryList []queries.Query, parseUrls bool) (*ParseResult, error) {
	data, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, err
	}
	result := ReadHtml(data, parseUrls)

	// Queries op uitvoeren
	source := queries.NewSource(result.Lowercased)
	var dataStr *string
	for _, query := range queryList {
		snippet := query.Execute(source)
		if snippet != nil {
			if dataStr == nil {
				str := string(data)
				dataStr = &str
			}
			apiResult := queries.NewResult(query, nil, nil, dataStr, result.Title, snippet)
			result.Results = append(result.Results, apiResult)
		}
	}

	return result, nil
}

func ReadHtml(data []byte, parseUrls bool) *ParseResult {
	reader := NewPositionReader(bytes.NewReader(data))

	lowercased := bytes.NewBuffer(make([]byte, 0, len(data)/2))

	head_depth := 0
	title_depth := 0
	var title string
	result := &ParseResult{Urls: make([]*url.URL, 0)}

	z := html.NewTokenizer(reader)

	ignore_depth := 0

	for {
		tt := z.Next()
		// Bytes worden hergebruikt, dus als ze
		// opgeslagen moeten worden is een kopie noodzakelijk
		switch tt {
		case html.ErrorToken:
			result.Lowercased = lowercased.Bytes()
			return result

		case html.TextToken:
			if ignore_depth == 0 && head_depth == 0 {
				lowercased.WriteByte(0)
				lowercased.Write(bytes.ToLower(z.Text()))

				/*endIndex := reader.Position - len(z.Buffered())
				str := z.Raw()
				startIndex := endIndex - len(str)

				// Nu alles verwijderen van previousEnd tot startIndex
				for i := previousEnd; i < startIndex; i++ {
					data[i] = 0
				}
				previousEnd = endIndex*/
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
				if parseUrls {
					key, val, moreAttr := z.TagAttr()
					for key != nil {

						if string(key) == "href" {

							attrUrl := ParseUrlFromHref(val)
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

func ParseUrlFromHref(href []byte) *url.URL {
	startIndex := 0
	endIndex := len(href) - 1

	for startIndex <= endIndex && (href[startIndex] == ' ' || href[startIndex] == '\t' || href[startIndex] == '\n' || href[startIndex] == '\f' || href[startIndex] == '\r') {
		startIndex++
	}

	for endIndex >= 0 && (href[endIndex] == ' ' || href[endIndex] == '\t' || href[endIndex] == '\n' || href[endIndex] == '\f' || href[endIndex] == '\r') {
		endIndex--
	}

	if startIndex >= endIndex+1 || startIndex >= len(href) || endIndex < 0 {
		return nil
	}

	u, err := url.Parse(string(href[startIndex : endIndex+1]))
	if err != nil {
		return nil
	}

	if len(u.Path) > 500 || len(u.RawQuery) > 500 {
		// Te lang, wrs gewoon onzinnige data
		return nil
	}

	u.Fragment = ""
	return u
}
