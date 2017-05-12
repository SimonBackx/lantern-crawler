package crawler

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxRecrawlDepth = 3

type Hostworker struct {
	Host   string // Domain without subdomains!
	Scheme string // Migrate automatically external domains?

	// Lijst met items die opnieuw moeten worden gecrawld met depth >= maxRecrawlDepth
	// Een item mag hier maximum 1 maand in verblijven (= vernieuwings interval)
	LowPriorityQueue *CrawlQueue

	// De queue bevat een lijst met items die nog nooit gecrawled werden met depth >= maxRecrawlDepth en
	// één voor één zullen worden gedownload. Deze hebben voorrang op recrawlen van pagina's met depth >= maxRecrawlDepth
	Queue *CrawlQueue

	// Lijst met items die een diepte hebben lager dan maxRecrawlDepth
	// Een item zou hier maximum 1 dag in mogen verblijven
	PriorityQueue *CrawlQueue

	// Lijst met items die te veel na elkaar mislukt zijn
	// Ze staan chonologisch gerangschikt. Degene die het langst geleden mislukt zijn,
	// staan vooraan in de wachtrij. Regelmatig controleren we of het eerste item ouder is dan 12 uur.
	// als dat zo is, halen we deze uit de lijst en verwijderen we deze uit AlreadyVisited om het opnieuw een kans te geven
	// om gecrawled te worden.
	FailedQueue *LeveledQueue

	// Hierin staan alle items die aanwezig zijn in Queue of PriorityQueue
	// OF items die niet aanwezig zijn in die queue's, maar die nog niet gedownload mogen worden
	// omdat item.NeedRecrawl false geeft. We
	// gaan deze af en toe uitkuisen en items die NeedsRecrawl true geven verwijderen
	// zo geraken we bewust pagina's kwijt waar nergens nog naar wordt verwezen
	AlreadyVisited map[string]*CrawlItem

	// Lijst met alle url's met diepte = 0. Deze staan cronologisch gerangschikt
	// van laatste gedownload naar meest recent gedownload
	IntroductionPoints *CrawlQueue

	Running         bool // Of goroutine loopt
	Sleeping        bool // Of deze worker in de sleeping queue aanwezig is
	InRecrawlList   bool
	RecrawlOnFinish bool // Enkel aanpassen of opvragen buiten de goroutine v/d worker

	FailStreak         int /// Aantal mislukte downloads na elkaar
	SucceededDownloads int // Aantal successvolle downloads (ooit)

	Client   *http.Client
	stop     chan struct{}
	NewItems popChannel
	crawler  *Crawler

	// Aantal requests die nog voltooid moeten worden
	// voor hij overweegt om naar slaapstand te gaan
	// als er andere domeinen 'wachten'
	sleepAfter int

	LatestCycle int
}

func (w *Hostworker) String() string {
	return w.Host
}

/**
 * Sla enkel de RecrawlQueue op. De AlreadyVisited maakt niet veel uit aangezien we deze uiteindelijk toch gaan opnieuw crawlen
 * als we de recrawl queue opnieuw crawlen.
 */
func (w *Hostworker) SaveToFile() {
	os.Mkdir("progress", 0777)
	file, err := os.Create("./progress/host_" + w.Host + ".txt")
	if err != nil {
		w.crawler.cfg.LogError(err)
		return
	}
	defer func() {
		file.Close()
	}()

	writer := bufio.NewWriter(file)
	w.SaveToWriter(writer)
	writer.Flush()
}

func NewHostWorkerFromFile(file *os.File, crawler *Crawler) *Hostworker {
	reader := bufio.NewReader(file)
	w := NewHostworker("", crawler)
	w.ReadFromReader(reader)
	return w
}

func (w *Hostworker) FillAlreadyVisited(q *CrawlQueue) {
	item := q.First
	for item != nil {
		uri, err := cleanURLPath(*item.URL)
		if err == nil {
			w.AlreadyVisited[uri] = item
		}
		item = item.Next
	}
}

func (w *Hostworker) GetRecrawlDuration() time.Duration {
	if w.IntroductionPoints.IsEmpty() {
		w.crawler.Panic("GetRecrawlDuration on worker with empty IntroductionPoints!")
		return time.Minute * 5
	}
	duration := time.Minute*30 - time.Since(*w.IntroductionPoints.First.LastDownload)

	return duration
}

func NewHostworker(host string, crawler *Crawler) *Hostworker {
	w := &Hostworker{
		Host:               host,
		Scheme:             "http",
		Queue:              NewCrawlQueue("Queue"),
		PriorityQueue:      NewCrawlQueue("Priority Queue"),
		LowPriorityQueue:   NewCrawlQueue("Low Priority Queue"),
		IntroductionPoints: NewCrawlQueue("Introduction points"),
		FailedQueue:        NewLeveledQueue(),

		AlreadyVisited: make(map[string]*CrawlItem),
		NewItems:       newPopChannel(),
		stop:           crawler.Stop,
		crawler:        crawler,
	}

	return w
}

func (w *Hostworker) EmptyPendingItems() {
	select {
	case q := <-w.NewItems:
		w.AddQueue(q)
	default:
		break
	}
}

func (w *Hostworker) WantsToGetUp() bool {
	if w.FailStreak > 20 && w.SucceededDownloads == 0 {
		// Passieve modus
		return false
	}

	result := !w.PriorityQueue.IsEmpty() || !w.Queue.IsEmpty() || !w.LowPriorityQueue.IsEmpty()
	if result {
		return true
	}

	// Misschien hebben we een item in de failed queue die er al uit mag komen?
	failedItem := w.FailedQueue.First()
	if failedItem != nil {
		if failedItem.NeedsRetry() {
			return true
		}
	}
	return false
}

func (w *Hostworker) AddQueue(q []*url.URL) {
	// Eerst nog overlopen op already visited, we kunnen dus niet rechtstreeks merge gebruiken
	for _, item := range q {
		w.NewReference(item, nil, false)
	}
}

/// Start een hercrawl cyclus. Voer dit enkel uit als de worker niet
/// 'aan' staat.
func (w *Hostworker) Recrawl() {
	w.LatestCycle++

	if w.crawler.cfg.LogRecrawlingEnabled {
		w.crawler.cfg.LogInfo("Recrawl initiated for " + w.String())
	}

	if !w.PriorityQueue.IsEmpty() {
		w.crawler.cfg.Log("warning", "Recrawl initiated before priority queue became empty")
	}

	item := w.IntroductionPoints.First
	for item != nil {
		item.Cycle = w.LatestCycle
		next := item.Next
		item.Remove()
		w.PriorityQueue.Push(item)

		item = next
	}
}

func (w *Hostworker) Run(client *http.Client) {
	defer func() {
		// Aangeven dat deze goroutine afgelopen is
		w.crawler.waitGroup.Done()

		// Onze crawler terug wakker maken om eventueel een nieuwe request op te starten
		w.crawler.WorkerEnded <- w
	}()

	if w.crawler.cfg.LogGoroutinesEnabled {
		w.crawler.cfg.LogInfo("Goroutine for host " + w.String() + " started")
	}

	w.Client = client

	// Snel horizontaal uitbreiden: neem laag getal
	if w.SucceededDownloads == 0 {
		w.sleepAfter = rand.Intn(5) + 1
	} else {
		w.sleepAfter = rand.Intn(20) + 6
	}

	for {
		select {
		case <-w.stop:
			return
		case q := <-w.NewItems:
			w.AddQueue(q)

		default:
			item := w.GetNextRequest()

			if item == nil {
				// queue is leeg
				return
			}

			// Onderstaande kansverdeling moet nog minder uniform gemaakt worden
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(4000)+4000))

			w.RequestStarted(item)
			w.Request(item)

			if w.sleepAfter <= 0 {
				// Meteen stoppen
				return
			}

		}
	}
}

func (w *Hostworker) Request(item *CrawlItem) {

	if request, err := http.NewRequest("GET", item.URL.String(), nil); err == nil {
		request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0")
		request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		request.Header.Add("Accept_Language", "en-US,en;q=0.5")
		request.Header.Add("Connection", "keep-alive")

		request.Close = true // Connectie weggooien
		request = request.WithContext(w.crawler.context)

		if response, err := w.Client.Do(request); err == nil {
			defer response.Body.Close()

			if response.StatusCode < 200 || response.StatusCode >= 300 {

				// Special exceptions
				if response.StatusCode == 429 {
					w.sleepAfter = -1
					w.crawler.cfg.Log("WARNING", "Too many requests for host "+w.String())
					w.RequestFailed(item)
					return
				}

				// ignore range: 400 - 406
				if response.StatusCode >= 400 && response.StatusCode <= 406 {
					w.RequestIgnored(item)
					return
				}

				if response.StatusCode >= 400 && response.StatusCode < 500 {
					// Lange tijd wachten voor het nog 2x opnieuw te proberen
					if item.FailCount < maxFailCount-2 {
						item.FailCount = maxFailCount - 2
					}

					w.RequestFailed(item)
				} else {
					// Retry eventually
					w.RequestFailed(item)
				}
				return
			}

			startTime := time.Now()

			// Maximaal 2MB (pagina's in darkweb zijn gemiddeld erg groot vanwege de afbeeldingen)
			if response.ContentLength > 2000000 {
				//w.crawler.cfg.LogInfo("Response: Content too long")
				// Too big
				// Eventueel op een ignore list zetten
				w.RequestIgnored(item)
				return
			}

			// Eerste 512 bytes lezen om zo de contentType te bepalen
			b, err := readFirstBytes(response.Body)
			if err != nil {
				// Er ging iets mis
				//w.crawler.cfg.LogError(err)
				w.RequestFailed(item)
				return
			}

			// Content type inlezen, als die niet goed zit stoppen...
			contentType := http.DetectContentType(b)
			//w.crawler.cfg.LogInfo("Detected Content-Type: " + contentType)

			if contentType != "text/html; charset=utf-8" {
				//w.crawler.cfg.LogInfo("Not a HTML file")
				// Op ignore list zetten
				w.RequestIgnored(item)
				return
			}

			firstReader := bytes.NewReader(b)

			// De twee readers terug samenvoegen
			reader := NewCountingReader(io.MultiReader(firstReader, response.Body), 2000000)
			if w.ProcessResponse(item, response, reader) {
				duration := time.Since(startTime)
				w.crawler.speedLogger.Log(duration, reader.Size)
			}

		} else {

			if response != nil && response.Body != nil {
				response.Body.Close()
			}

			str := err.Error()
			if strings.Contains(str, "SOCKS5") {
				// Er is iets mis met de proxy,
				// zal zich normaal uatomatisch herstellen, maar
				// we stoppen even met deze crawler
				w.sleepAfter = -1

				// Even negeren
				w.FailStreak--
			} else if strings.Contains(str, "Client.Timeout") {
				w.crawler.speedLogger.LogTimeout()
			} else if strings.Contains(str, "timeout") {
				w.crawler.speedLogger.LogTimeout()
			} else if strings.Contains(str, "stopped after 10 redirects") {
				w.RequestIgnored(item)
				return
			} else if strings.Contains(str, "server gave HTTP response to HTTPS client") {
				w.Scheme = "http"
				item.URL.Scheme = "http"
			} else if strings.Contains(str, "context canceled") {
				// Negeer failcount bij handmatige cancel
				item.FailCount--
			}
			w.RequestFailed(item)

			// (Client.Timeout exceeded while awaiting headers)

			// tor proxy niet bereikbaar:
			// Get http://www.scoutswetteren.be: net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)

			// timeout awaiting response headers
			// request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)
		}
	} else {

		w.RequestFailed(item)
	}

}

func (w *Hostworker) ProcessResponse(item *CrawlItem, response *http.Response, reader io.Reader) bool {
	// Doorgeven aan parser
	result, err := Parse(reader, w.crawler.Queries)

	if err != nil {
		if err.Error() == "Reader reached maximum bytes!" {
			w.RequestIgnored(item)
			return false
		}

		w.RequestFailed(item)
		return false
	}

	if response.Request.URL.Scheme == "https" {
		w.Scheme = "https"
	} else if response.Request.URL.Scheme == "http" {
		w.Scheme = "http"
	}

	item.URL = response.Request.URL

	// Save results
	if len(result.Results) > 0 {
		host := w.String()
		urlString := item.URL.String()

		for _, apiResult := range result.Results {
			apiResult.Host = &host
			apiResult.Url = &urlString
			if apiResult.Title == nil {
				apiResult.Title = &host
			}

			w.crawler.ApiController.SaveResult(apiResult)
		}
	}

	workerResult := NewWorkerResult()

	if result.Links != nil {
		for _, link := range result.Links {
			// todo: spaties / tabs verwijderen uit url voor of einde
			u := link.Href

			// Convert links to absolute url
			ResolveReferenceNoCopy(response.Request.URL, u)

			// Url moet absoluut zijn
			if !u.IsAbs() {
				panic("Resolve reference didn't make absolute")
				break
			}

			if !strings.HasPrefix(u.Scheme, "http") {
				break
			}

			// Host opspliten in subdomein en domein
			domains := strings.Split(u.Host, ".")
			if len(domains) < 2 {
				break
			}

			if w.crawler.cfg.OnlyOnion {
				tld := domains[len(domains)-1]
				if tld != "onion" {
					break
				}

				domain := domains[len(domains)-2]

				if len(domain) != 22 {
					// todo: ondersteuning voor tor subdomains toevoegen!
					// Ongeldig -> verwijder alle ongeldige characters (tor browser doet dit ook)
					reg := regexp.MustCompile("[^a-zA-Z2-7]+")
					domain = reg.ReplaceAllString(domain, "")
					if len(domain) != 22 {
						break
					}
					// Terug samenvoegen
					domains[len(domains)-2] = domain
					u.Host = strings.Join(domains, ".")
				}
			} else {
				if len(domains[len(domains)-1]) < 2 {
					// tld te kort
					break
				}

				if len(domains[len(domains)-2]) < 1 {
					// domain te kort
					break
				}
			}

			if w.crawler.GetDomainForUrl(u, domains) == w.Host {
				// Interne URL's meteen verwerken
				w.NewReference(u, item, true)
			} else {
				workerResult.Append(u)
			}
		}
	}

	// Kritieke move operatie uitvoeren noodzakelijk?
	if w.crawler.GetDomainForUrl(item.URL, strings.Split(item.URL.Hostname(), ".")) != w.Host {
		// Negeren vanaf nu voor deze worker
		w.RequestIgnored(item)

		// Doorgeven aan crawler en aan juiste worker bezorgen voor verdere afhandeling?
		workerResult.Append(item.URL)
		w.crawler.WorkerResult <- workerResult

		return false
	}

	// Resultaat doorgeven aan Crawler
	if len(workerResult.Links) > 0 {
		w.crawler.WorkerResult <- workerResult
	}

	w.RequestFinished(item)
	return true
}

func (w *Hostworker) RequestStarted(item *CrawlItem) {
	w.sleepAfter--

	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request started. %v", item.URL.String()))
	now := time.Now()
	item.LastDownloadStarted = &now
}

func (w *Hostworker) RequestFinished(item *CrawlItem) {
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request finished. %v", item.URL.String()))
	w.FailStreak = 0
	w.SucceededDownloads++

	if item.Depth == 0 {
		// Introduction point toevoegen
		if w.IntroductionPoints.IsEmpty() {
			w.IntroductionPoints.Push(item)

			// Crawler verwittigen zodat we op de recrawl lijst komen
			w.crawler.WorkerIntroduction <- w
		} else {
			if w.IntroductionPoints.Length < 10 {
				w.IntroductionPoints.Push(item)
			}
		}
	}

	if item.FailCount > 0 {
		item.FailCount = 0
	}

	now := time.Now()
	item.LastDownload = &now
	item.LastDownloadStarted = nil
}

func (w *Hostworker) RequestIgnored(item *CrawlItem) {
	item.Ignore = true
}

func (w *Hostworker) RequestFailed(item *CrawlItem) {
	item.FailCount++

	if item.FailCount == 2 {
		// 2e poging is ook mislukt
		w.FailStreak++

		if w.FailStreak > 3 {
			// Meteen stoppen
			w.sleepAfter = -1
		}
	}

	if !item.IsUnavailable() {
		// We wagen nog een poging binnen een uurtje
		// Toevoegen aan failqueue
		w.FailedQueue.Push(item, item.FailCount)
	}
}

func (w *Hostworker) GetNextRequest() *CrawlItem {
	f := w.FailedQueue.Pop()

	if f != nil {
		return f
	}

	if !w.PriorityQueue.IsEmpty() {
		return w.PriorityQueue.Pop()
	}

	if !w.Queue.IsEmpty() {
		return w.Queue.Pop()
	}

	if w.LowPriorityQueue.IsEmpty() {
		return nil
	}

	return w.LowPriorityQueue.Pop()
}

func cleanURLPath(u url.URL) (string, error) {
	// todo!!!: verbruikt te veel geheugen
	//
	u.Scheme = "" // todo: checken?

	normalized := purell.NormalizeURL(&u,
		purell.FlagDecodeUnnecessaryEscapes|
			purell.FlagUppercaseEscapes|
			purell.FlagEncodeNecessaryEscapes|
			purell.FlagRemoveDefaultPort|
			purell.FlagRemoveEmptyQuerySeparator|
			purell.FlagRemoveFragment|
			purell.FlagRemoveEmptyPortSeparator|
			purell.FlagRemoveTrailingSlash)

	// Allocatie vermijden???
	clean, err := url.ParseRequestURI(normalized)

	if err != nil {
		return "", err
	}

	return clean.String(), nil
}

func (w *Hostworker) VisitedItem(item *CrawlItem) {
	uri, err := cleanURLPath(*item.URL)
	if err != nil {
		w.crawler.cfg.LogError(err)
		return
	}

	w.AlreadyVisited[uri] = item
}

/**
 * Als internal = false mag sourceItem = nil
 */
func (w *Hostworker) NewReference(foundUrl *url.URL, sourceItem *CrawlItem, internal bool) (*CrawlItem, error) {
	uri, err := cleanURLPath(*foundUrl)
	if err != nil {
		w.crawler.cfg.LogError(err)
		return nil, err
	}

	if !foundUrl.IsAbs() {
		return nil, nil
	}

	item, found := w.AlreadyVisited[uri]
	if !found {
		item = NewCrawlItem(foundUrl)
		if internal {
			item.Cycle = sourceItem.Cycle
		} else {
			// New introduction point
			item.Cycle = w.LatestCycle

			// Schema meteen juist zetten
			item.URL.Scheme = w.Scheme
		}

		w.AlreadyVisited[uri] = item
	} else {
		if item.IsUnavailable() {
			// Deze url is onbereikbaar, ofwel geen HTML bestand
			// dat weten we omdat we deze al eerder hebben gecrawled
			return item, nil
		}
	}

	// Depth aanpassen
	if !internal {
		// Referentie vanaf een ander domein
		item.Depth = 0

	} else {
		if !found || item.Depth >= sourceItem.Depth+1 {
			item.Depth = sourceItem.Depth + 1
		}
	}

	if internal && item.Cycle < sourceItem.Cycle {
		// Als een nieuwere cycle refereert naar deze pagina, dan kan
		// die de depth verhogen. Dit kan slechts één keer gebeuren,
		// aangezien hierna de cycle terug wordt gelijk gesteld
		// Daarna kan de depth enkel nog verlagen tot de volgende cycle
		// Op die manier houdt het systeem rekening met verloren / gewijzigde referenties

		item.Depth = sourceItem.Depth + 1
	}

	if item.Depth < maxRecrawlDepth && (item.Queue == w.Queue || item.Queue == w.LowPriorityQueue) {
		// Dit item staat nog in de gewone queue, maar heeft nu wel prioriteit
		// we verplaatsen het
		item.Remove()
		w.PriorityQueue.Push(item)

	} else if item.Queue == nil && (!found || (internal && item.Cycle < sourceItem.Cycle)) {
		// Recrawl enkel toelaten als we dit item nog niet gevonden hebben
		// of we hebben het wel al gevonden en het is een interne link afkomstig van een
		// hogere cycle (recrawl). Externe links die we al gecrawled hebben
		// negeren we, die staan in de introduction queue

		if item.Depth < maxRecrawlDepth {
			w.PriorityQueue.Push(item)
		} else {
			if !found {
				w.Queue.Push(item)
			} else {
				w.LowPriorityQueue.Push(item)
			}
		}
	}

	// Cycle aanpassen
	if internal && item.Cycle < sourceItem.Cycle {
		item.Cycle = sourceItem.Cycle
	}

	return item, nil
}

//
//
// Saving functions
//
//

func (w *Hostworker) ReadFromReader(reader *bufio.Reader) {
	// Eerst de basis gegevens:
	line, _, _ := reader.ReadLine()
	if len(line) == 0 {
		return
	}
	str := string(line)
	parts := strings.Split(str, "	")
	if len(parts) != 5 {
		return
	}

	w.Host = parts[0]
	w.Scheme = parts[1]

	num, err := strconv.Atoi(parts[2])
	if err != nil {
		fmt.Println("Invalid failstreak")
		return
	}
	w.FailStreak = num

	num, err = strconv.Atoi(parts[3])
	if err != nil {
		fmt.Println("Invalid SucceededDownloads")
		return
	}
	w.SucceededDownloads = num

	num, err = strconv.Atoi(parts[4])
	if err != nil {
		fmt.Println("Invalid LatestCycle")
		return
	}
	w.LatestCycle = num

	w.IntroductionPoints.ReadFromReader(reader)
	w.PriorityQueue.ReadFromReader(reader)
	w.Queue.ReadFromReader(reader)
	w.LowPriorityQueue.ReadFromReader(reader)
	w.FailedQueue.ReadFromReader(reader)

	line, _, _ = reader.ReadLine()
	for len(line) > 0 {
		str = string(line)
		item := NewCrawlItemFromString(&str)
		if item != nil {
			w.VisitedItem(item)
		} else {
			fmt.Println("Invalid item: " + str)
		}
		line, _, _ = reader.ReadLine()
	}

	// Alle queue's toevoegen aan already visited
	var item *CrawlItem
	item = w.IntroductionPoints.First
	for item != nil {
		w.VisitedItem(item)
		item = item.Next
	}
	item = w.PriorityQueue.First
	for item != nil {
		w.VisitedItem(item)
		item = item.Next
	}
	item = w.Queue.First
	for item != nil {
		w.VisitedItem(item)
		item = item.Next
	}
	item = w.LowPriorityQueue.First
	for item != nil {
		w.VisitedItem(item)
		item = item.Next
	}

	for _, queue := range w.FailedQueue.Levels {
		item = queue.First
		for item != nil {
			w.VisitedItem(item)
			item = item.Next
		}
	}

}

func (w *Hostworker) SaveToWriter(writer *bufio.Writer) {
	str := fmt.Sprintf(
		"%s	%s	%v	%v	%v",
		w.Host,
		w.Scheme,
		w.FailStreak,
		w.SucceededDownloads,
		w.LatestCycle,
	)
	writer.WriteString(str)
	writer.WriteString("\n")

	w.IntroductionPoints.SaveToWriter(writer)
	w.PriorityQueue.SaveToWriter(writer)
	w.Queue.SaveToWriter(writer)
	w.LowPriorityQueue.SaveToWriter(writer)
	w.FailedQueue.SaveToWriter(writer)

	// Nu de rest opslaan
	for _, value := range w.AlreadyVisited {
		if value.Queue == nil {
			// Staat in geen andere queue
			writer.WriteString(value.SaveToString())
			writer.WriteString("\n")
		}
	}
}

func (w *Hostworker) IsEqual(b *Hostworker) bool {
	if w.Host != b.Host {
		return false
	}

	if w.Scheme != b.Scheme {
		return false
	}

	if w.FailStreak != b.FailStreak {
		return false
	}

	if w.SucceededDownloads != b.SucceededDownloads {
		return false
	}

	if w.LatestCycle != b.LatestCycle {
		return false
	}

	if !w.IntroductionPoints.IsEqual(b.IntroductionPoints) {
		return false
	}

	if !w.PriorityQueue.IsEqual(b.PriorityQueue) {
		return false
	}

	if !w.Queue.IsEqual(b.Queue) {
		return false
	}

	if !w.LowPriorityQueue.IsEqual(b.LowPriorityQueue) {
		return false
	}

	if !w.FailedQueue.IsEqual(b.FailedQueue) {
		return false
	}

	// todo: already visited checken!
	if len(w.AlreadyVisited) != len(b.AlreadyVisited) {
		return false
	}

	for key, value := range w.AlreadyVisited {
		other, found := b.AlreadyVisited[key]
		if !found {
			return false
		}
		if !value.IsEqual(other) {
			return false
		}
	}

	return true
}
