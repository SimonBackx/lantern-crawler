package crawler

import (
	"net/url"
	"testing"
	"time"
)

func TestCrawlItem(test *testing.T) {
	u, _ := url.Parse("/websitepage")
	item := NewCrawlItem(u)
	item.Depth = 23
	item.Cycle = 978655
	item.FailCount = 3
	now := time.Now()

	item.LastDownloadStarted = &now
	item.LastDownload = &now

	str := item.SaveToString()

	itemCopy := NewCrawlItemFromString(&str)

	if !item.IsEqual(itemCopy) {
		test.Log("Save not equal")
		test.Log(str)
		test.Log(itemCopy.SaveToString())
		test.Fail()
	}

}
