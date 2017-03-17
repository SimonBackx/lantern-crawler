package crawler

import (
	"fmt"
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

func (item *CrawlItem) String() string {
	return fmt.Sprintf("Item - URL = %v;", item.URL.EscapedPath())
}
