package crawler

import (
	"fmt"
	"net/url"
	"time"
)

type CrawlItem struct {
	URL   *url.URL
	Depth int

	// Laatste tijdstip dat deze pagina successvol werd gedownload en al haar link's werden verwerkt
	LastDownload *time.Time

	// Laatste tijdstip waarop een URL gevonden werd naar deze pagina
	LastReference *time.Time

	Method string
	Body   *string

	// Positie in de queue (enkel aanpassen in CrawlQueue!)
	Next     *CrawlItem
	Previous *CrawlItem
	Queue    *CrawlQueue
}

func NewCrawlItem(URL *url.URL) *CrawlItem {
	return &CrawlItem{URL: URL}
}

func (i *CrawlItem) String() string {
	return fmt.Sprintf("URL = %v", i.URL.EscapedPath())
}

func (i *CrawlItem) NeedsRecrawl() bool {
	if i.LastDownload == nil {
		return false
	}

	// Todo: geavanceerder maken
	return time.Since(*i.LastDownload) > 12*time.Hour
}

/**
 * Remove is noodzakelijk voor wanneer de depth aangepast wordt
 * @param  {[type]} i *CrawlItem)   Remove( [description]
 * @return {[type]}   [description]
 */
func (i *CrawlItem) Remove() {
	if i.Queue != nil {
		i.Queue.Remove(i)
	}
}
