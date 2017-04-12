package crawler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const crawlItemTimeFormat = "2006-01-02 15:04:05.999999999"

type CrawlItem struct {
	URL           *url.URL
	Depth         int
	DownloadCount int

	// Laatste tijdstip dat deze pagina successvol werd gedownload en al haar link's werden verwerkt
	LastDownload *time.Time

	// Laatste tijdstip waarop een URL gevonden werd naar deze pagina
	LastReference    *time.Time
	LastReferenceURL *url.URL

	// Positie in de queue (enkel aanpassen in CrawlQueue!)
	Next     *CrawlItem
	Previous *CrawlItem
	Queue    *CrawlQueue
}

func NewCrawlItem(URL *url.URL) *CrawlItem {
	return &CrawlItem{URL: URL}
}

func NewCrawlItemFromString(str *string) *CrawlItem {
	parts := strings.Split(*str, ",")
	if len(parts) != 4 {
		return nil
	}
	url, err := url.ParseRequestURI(parts[0])
	if err != nil {
		return nil
	}

	depth, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}

	// Als parse mislukt -> nil (dan was het wrs ook nil)
	download, _ := time.Parse(crawlItemTimeFormat, parts[2])
	reference, _ := time.Parse(crawlItemTimeFormat, parts[3])

	return &CrawlItem{
		URL:           url,
		Depth:         depth,
		LastDownload:  &download,
		LastReference: &reference,
	}
}

func (i *CrawlItem) String() string {
	return i.URL.EscapedPath()
}

func (i *CrawlItem) NeedsRecrawl() bool {
	if i.LastDownload == nil {
		return false
	}

	// Todo: geavanceerder maken

	t := 60 * time.Second
	if i.Depth == 0 {
		// Introduction points moeten even langer wachten voor ze opnieuw mogen worden gerecrawld
		t = 100 * time.Second
	}

	answer := time.Since(*i.LastDownload) > t //time.Hour

	return answer
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

func TimeToString(time *time.Time) string {
	if time == nil {
		return ""
	}
	return time.Format(crawlItemTimeFormat)
}

func (i *CrawlItem) SaveToString() string {
	return fmt.Sprintf("%v,%v,%s,%s,%s", i.URL.EscapedPath(), i.Depth, TimeToString(i.LastDownload), TimeToString(i.LastReference), i.LastReferenceURL)
}
