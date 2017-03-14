package crawler

import (
	"net/url"
)

type CrawlItem struct {
	URL    *url.URL
	Method string
	Body   *string
	Next   *CrawlItem
}

func NewCrawlItem(URL *url.URL) *CrawlItem {
	return &CrawlItem{URL: URL}
}
