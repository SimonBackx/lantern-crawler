package crawler

import (
	"net/url"
)

type CrawlItem struct {
	URL     *url.URL
	Method  string
	Body    *string
	Website *Website
	Next    *CrawlItem
}

func NewCrawlItem(URL *url.URL, Website *Website) *CrawlItem {
	return &CrawlItem{URL: URL, Website: Website}
}
