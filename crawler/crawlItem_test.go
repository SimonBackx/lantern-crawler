package crawler

import (
	"net/url"
	"testing"
	"time"
)

func TestCrawlItem(test *testing.T) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})
	worker1 := NewHostworker("test.com", crawler)

	u, _ := url.Parse("https://www.test.com/websitepage")
	item, _ := worker1.NewReference(u, nil, false)
	item.Depth = 23
	item.Cycle = 978655
	item.FailCount = 3
	now := time.Now()

	item.LastDownloadStarted = &now
	item.LastDownload = &now

	item.Subdomain.Index = 0
	str := item.SaveToString()

	itemCopy := NewCrawlItemFromString(&str, []*Subdomain{item.Subdomain})

	if !item.IsEqual(itemCopy) {
		test.Log("Save not equal")
		test.Log(str)
		test.Log(itemCopy.SaveToString())
		test.Fail()
	}

}
