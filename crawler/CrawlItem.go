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
const maxFailCount = 6

type CrawlItem struct {
	URL    *url.URL
	Depth  int
	Cycle  int
	Ignore bool

	// Aantal opeenvolgende mislukte downloads
	FailCount           int
	LastDownloadStarted *time.Time

	// Laatste tijdstip dat deze pagina successvol werd gedownload en al haar link's werden verwerkt
	// nodig voor intrudction points
	LastDownload *time.Time

	// Positie in de queue (enkel aanpassen in CrawlQueue!)
	Next     *CrawlItem
	Previous *CrawlItem
	Queue    *CrawlQueue

	Subdomain *Subdomain
}

func (c *CrawlItem) IsEqual(b *CrawlItem) bool {
	if b == nil || c == nil {
		return false
	}

	if c.URL.String() != b.URL.String() {
		return false
	}

	if c.Depth != b.Depth {
		return false
	}

	if c.Cycle != b.Cycle {
		return false
	}

	if c.Ignore != b.Ignore {
		return false
	}

	if c.FailCount != b.FailCount {
		return false
	}

	if !(c.LastDownloadStarted == nil && b.LastDownloadStarted == nil) && (c.LastDownloadStarted == nil || b.LastDownloadStarted == nil || c.LastDownloadStarted.Equal(*b.LastDownloadStarted)) {
		return false
	}

	if !(c.LastDownload == nil && b.LastDownload == nil) && (c.LastDownload == nil || b.LastDownload == nil || c.LastDownload.Equal(*b.LastDownload)) {
		return false
	}

	if !(c.Subdomain == nil && b.Subdomain == nil) && (c.Subdomain == nil || b.Subdomain == nil || c.Subdomain.Url.String() != b.Subdomain.Url.String()) {
		return false
	}

	return true
}

func NewCrawlItem(URL *url.URL) *CrawlItem {
	if URL.IsAbs() {
		panic("Is absolute crawl item " + URL.String())
	}
	return &CrawlItem{URL: URL}
}

func NewCrawlItemFromString(str *string, subdomains []*Subdomain) *CrawlItem {
	parts := strings.Split(*str, "	")
	if len(parts) != 8 {
		fmt.Println("Ongeldig aantal tabs")
		return nil
	}

	u, err := url.Parse(parts[0])
	if err != nil || u.IsAbs() {
		fmt.Println("ongeldige url")
		return nil
	}

	depth, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("ongeldige depth")
		return nil
	}

	cycle, err := strconv.Atoi(parts[2])
	if err != nil {
		fmt.Println("ongeldige cycle")
		return nil
	}

	ignore := (parts[3] == "true")
	if parts[3] != "false" && parts[3] != "true" {
		fmt.Println("ongeldige ignore")
		return nil
	}

	failCount, err := strconv.Atoi(parts[4])
	if err != nil {
		fmt.Println("ongeldige failCount")
		return nil
	}

	// Als parse mislukt -> nil (dan was het wrs ook nil)
	var d *time.Time
	download, err := time.Parse(crawlItemTimeFormat, parts[5])
	if len(parts[5]) > 0 && err != nil {
		fmt.Println("ongeldige download datum")
		return nil
	}

	if err == nil {
		d = &download
	}

	var ds *time.Time
	downloadStarted, err := time.Parse(crawlItemTimeFormat, parts[6])
	if len(parts[6]) > 0 && err != nil {
		fmt.Println("ongeldige download started datum")
		return nil
	}

	if err == nil {
		ds = &downloadStarted
	}

	if subdomains == nil {
		return &CrawlItem{
			URL:                 u,
			Depth:               depth,
			Cycle:               cycle,
			Ignore:              ignore,
			FailCount:           failCount,
			LastDownload:        d,
			LastDownloadStarted: ds,
		}
	}

	subdomainIndex, err := strconv.Atoi(parts[7])
	if err != nil {
		fmt.Println("ongeldige subdomainIndex")
		return nil
	}

	if len(subdomains) <= subdomainIndex {
		fmt.Println("subdomain niet gevonden")
		return nil
	}
	subdomain := subdomains[subdomainIndex]

	item := &CrawlItem{
		URL:                 u,
		Depth:               depth,
		Cycle:               cycle,
		Ignore:              ignore,
		FailCount:           failCount,
		LastDownload:        d,
		LastDownloadStarted: ds,
		Subdomain:           subdomain,
	}

	subdomain.AlreadyVisted[cleanURLPath(u)] = item

	return item
}

func (i *CrawlItem) String() string {
	if i.Subdomain == nil {
		return i.URL.String()
	}
	return i.Subdomain.Url.String() + i.URL.String()
}

func (i *CrawlItem) IsUnavailable() bool {
	return i.FailCount > maxFailCount || i.Ignore
}

func (i *CrawlItem) FakeRetry() {
	n := time.Now()
	i.LastDownloadStarted = &n
	i.FailCount = 1
}

func (i *CrawlItem) NeedsRetry() bool {
	if i.LastDownloadStarted == nil {
		return false
	}

	if i.FailCount == 1 {
		// Meteen opnieuw proberen
		return true
	}

	// n = maxFailCount - 1
	// a^1 + a^2 + a^3 + a^4 + a^5 + a^6 + a^7 + a^8 + a^9 + a^n = 44640 minuten (= 1 maand)
	// => a = 2.79 voor (n = 10)
	// => a = 8.3 (n = 5)
	// Het duurt 1 maand voor een request verwijderd wordt als we deze formule gebruiken
	// Deze exponentiele retry tijd is enkel mogelijk dankzij de leveledQueue
	// Op die manier kunnen we het sorteren van items vermijden
	answer := time.Since(*i.LastDownloadStarted) > time.Duration(math.Pow(8.3, float64(i.FailCount-1)))*time.Minute //time.Hour

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
	var index int
	if i.Subdomain != nil {
		index = i.Subdomain.Index
	}
	if i.URL.IsAbs() {
		fmt.Println("CrawlItem url became absolute")
	}
	return fmt.Sprintf("%s	%v	%v	%v	%v	%s	%s	%v", i.URL, i.Depth, i.Cycle, i.Ignore, i.FailCount, TimeToString(i.LastDownload), TimeToString(i.LastDownloadStarted), index)
}
