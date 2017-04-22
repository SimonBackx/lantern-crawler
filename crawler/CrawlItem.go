package crawler

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const crawlItemTimeFormat = "2006-01-02-15:04:05.999999999"

type CrawlItem struct {
	URL           *url.URL
	Depth         int
	DownloadCount int

	// Aantal opeenvolgende mislukte downloads
	FailCount           int
	LastDownloadStarted *time.Time

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
	parts := strings.Split(*str, "	")
	if len(parts) != 5 {
		fmt.Println("Ongeldig aantal tabs")
		return nil
	}

	url, err := url.ParseRequestURI(parts[0])
	if err != nil {
		fmt.Println("ongeldige url")
		return nil
	}

	depth, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("ongeldige depth")
		return nil
	}

	downloadCount, err := strconv.Atoi(parts[2])
	if err != nil {
		fmt.Println("ongeldige download count")
		return nil
	}

	// Als parse mislukt -> nil (dan was het wrs ook nil)
	download, err := time.Parse(crawlItemTimeFormat, parts[3])
	if len(parts[3]) > 0 && err != nil {
		fmt.Println("ongeldige download datum")
		return nil
	}

	reference, err := time.Parse(crawlItemTimeFormat, parts[4])

	if len(parts[4]) > 0 && err != nil {
		fmt.Println("ongeldige reference datum")
		return nil
	}

	return &CrawlItem{
		URL:           url,
		Depth:         depth,
		DownloadCount: downloadCount,
		LastDownload:  &download,
		LastReference: &reference,
	}
}

func (i *CrawlItem) String() string {
	return i.URL.EscapedPath()
}

func (i *CrawlItem) IsUnavailable() bool {
	return i.FailCount > 10
}

func (i *CrawlItem) NeedsRetry() bool {
	if i.LastDownloadStarted == nil {
		return false
	}

	// a^1 + a^2 + a^3 + a^4 + a^5 + a^6 + a^7 + a^8 + a^9 + a^10 = 44640 minuten (= 1 maand)
	// => a = 2.79
	// Het duurt 1 maand voor een request verwijderd wordt als we deze formule gebruiken
	// Deze exponentiele retry tijd is enkel mogelijk dankzij de leveledQueue
	// Op die manier kunnen we het sorteren van items vermijden
	answer := time.Since(*i.LastDownloadStarted) > time.Duration(math.Pow(2.79, float64(i.FailCount)))*time.Minute //time.Hour

	return answer
}

func (i *CrawlItem) NeedsRecrawl() bool {
	if i.LastDownload == nil || i.IsUnavailable() {
		return false
	}

	// Een pagina maar opnieuw crawlen na 30 minuten. Dit moet altijd
	// veel lager liggen dan het recrawl interval!
	return time.Since(*i.LastDownload) > 30*time.Minute
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
	return fmt.Sprintf("%s	%v	%v	%s	%s", i.URL, i.Depth, i.DownloadCount, TimeToString(i.LastDownload), TimeToString(i.LastReference))
}
