package crawler

import (
	"bufio"
	"bytes"
	"github.com/PuerkitoBio/purell"
	"github.com/SimonBackx/lantern-crawler/queries"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const maxRecrawlDepth = 2

type Hostworker struct {
	Host string

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
	writer.WriteString(w.String())
	writer.WriteString("\n")

	w.IntroductionPoints.SaveToWriter(writer)
	writer.WriteString("\n")

	w.PriorityQueue.SaveToWriter(writer)
	writer.WriteString("\n")

	// Failed queue gewoon toevoegen aan queue
	w.Queue.SaveToWriter(writer)
	w.FailedQueue.SaveToWriter(writer)

	writer.WriteString("\n")

	w.LowPriorityQueue.SaveToWriter(writer)

	writer.Flush()
}

func NewHostWorkerFromFile(file *os.File, crawler *Crawler) *Hostworker {
	reader := bufio.NewReader(file)
	host, _, err := reader.ReadLine()
	if len(host) == 0 || err != nil {
		crawler.cfg.LogInfo("Invalid file while loading host file")
		return nil
	}

	w := NewHostworker(string(host), crawler)
	w.IntroductionPoints.ReadFromReader(reader)
	w.FillAlreadyVisited(w.IntroductionPoints)

	w.PriorityQueue.ReadFromReader(reader)
	w.FillAlreadyVisited(w.PriorityQueue)

	w.Queue.ReadFromReader(reader)
	w.FillAlreadyVisited(w.Queue)

	w.LowPriorityQueue.ReadFromReader(reader)
	w.FillAlreadyVisited(w.LowPriorityQueue)

	return w
}

func (w *Hostworker) FillAlreadyVisited(q *CrawlQueue) {
	item := q.First
	for item != nil {
		uri, err := cleanURLPath(item.URL)
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
		Queue:              NewCrawlQueue("Queue"),
		PriorityQueue:      NewCrawlQueue("Priority Queue"),
		LowPriorityQueue:   NewCrawlQueue("Low Priority Queue"),
		IntroductionPoints: NewCrawlQueue("Intoroduction points"),
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
	return !w.PriorityQueue.IsEmpty() || !w.Queue.IsEmpty() || !w.LowPriorityQueue.IsEmpty()
}

func (w *Hostworker) AddQueue(q *CrawlQueue) {
	// Eerst nog overlopen op already visited, we kunnen dus niet rechtstreeks merge gebruiken
	item := q.First
	for item != nil {
		w.NewReference(item.URL, nil, item.LastReferenceURL, nil)
		item = item.Next
	}
}

/// Start een hercrawl cyclus. Voer dit enkel uit als de worker niet
/// 'aan' staat.
func (w *Hostworker) Recrawl() {
	if w.crawler.cfg.LogRecrawlingEnabled {
		w.crawler.cfg.LogInfo("Recrawl initiated for " + w.String())
	}

	item := w.IntroductionPoints.First
	for item != nil {
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
	w.sleepAfter = rand.Intn(20) + 1

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
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(3000)+4000))

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
				switch response.StatusCode {
				case 429:
					w.sleepAfter = -1
					w.crawler.cfg.Log("WARNING", "Too many requests for host "+w.String())
					w.RequestFailed(item)
					return
				}

				if response.StatusCode >= 400 && response.StatusCode < 500 {
					item.Ignore = true
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
				item.Ignore = true
				return
			}

			// Eerste 512 bytes lezen om zo de contentType te bepalen
			b, err := readFirstBytes(response.Body)
			if err != nil {
				// Er ging iets mis
				w.crawler.cfg.LogError(err)
				w.RequestFailed(item)
				return
			}

			// Content type inlezen, als die niet goed zit stoppen...
			contentType := http.DetectContentType(b)
			//w.crawler.cfg.LogInfo("Detected Content-Type: " + contentType)

			if contentType != "text/html; charset=utf-8" {
				//w.crawler.cfg.LogInfo("Not a HTML file")
				// Op ignore list zetten
				item.Ignore = true
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
			} else if strings.Contains(str, "(Client.Timeout exceeded while awaiting headers)") {
				w.crawler.speedLogger.Timeouts++
			} else if strings.Contains(str, "timeout awaiting response headers") {
				w.crawler.speedLogger.Timeouts++
			} else if strings.Contains(str, "stopped after 10 redirects") {
				item.Ignore = true
			} else if strings.Contains(str, "context canceled") {
				// Negeer failcount bij handmatige cancel
				item.FailCount--
			} else {
				w.crawler.cfg.LogError(err)
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
	// Todo: status code controleren en dit correct afhandelen.

	requestUrl := *response.Request.URL
	urlRef := &requestUrl

	// Doorgeven aan parser
	result, err := Parse(reader, w.crawler.Queries)

	if err != nil {
		if err.Error() == "Reader reached maximum bytes!" {
			w.crawler.cfg.LogError(err)
			item.Ignore = true
		}

		w.RequestFailed(item)
		return false
	}

	for _, query := range result.Queries {
		apiResult := queries.NewResult(query, requestUrl.String(), result.Document)
		w.crawler.cfg.LogInfo("Found " + query.String() + " at " + w.String() + item.String())
		w.crawler.ApiController.SaveResult(apiResult)
	}

	workerResult := NewWorkerResult()
	workerResult.Source = urlRef

	if result.Links != nil {
		for _, link := range result.Links {
			// Convert links to absolute url
			u := urlRef.ResolveReference(&link.Href)

			// Url moet absoluut zijn
			if u == nil || !u.IsAbs() {
				break
			}

			if !strings.HasPrefix(u.Scheme, "http") {
				break
			}

			// Alle invalid characters verwijderen
			reg := regexp.MustCompile("[^0-9a-zA-Z,.\\-!/()=?`*;:_{}[]\\|~]+")
			u, err := url.Parse(reg.ReplaceAllString(u.String(), ""))
			if err != nil {
				break
			}

			// Normaisaties toepassen
			normalized := purell.NormalizeURL(u,
				purell.FlagsSafe|purell.FlagRemoveFragment)
			u, err = url.ParseRequestURI(normalized)

			if err != nil {
				w.crawler.cfg.LogError(err)
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
					w.crawler.cfg.LogInfo("Fixed invalid onion url")
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

			if u.Hostname() == w.Host {
				// Interne URL's meteen verwerken
				w.NewReference(u, &item.Depth, urlRef, item)
			} else {
				workerResult.Append(u)
			}
		}
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

	// DownloadCount verhogen ongeacht gelukt of mislukt (is noodzakelijk detectie verdwenen referenties)
	item.DownloadCount++

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
			w.IntroductionPoints.Push(item)
		}
	}

	if item.FailCount > 0 {
		item.FailCount = 0
		w.crawler.speedLogger.LogSuccessfulRetry()
	}

	if item.LastDownload != nil {
		// Recrawl
		w.crawler.speedLogger.RecrawlCount++
	}

	now := time.Now()
	item.LastDownload = &now

}

func (w *Hostworker) RequestFailed(item *CrawlItem) {
	item.FailCount++

	if item.FailCount == 2 {
		// 2e poging is ook mislukt
		w.FailStreak++

		if w.FailStreak > 10 {
			// Meteen stoppen
			w.sleepAfter = -1
			w.crawler.cfg.LogInfo("Failstreak voor " + w.String())
		}
	}

	if !item.IsUnavailable() {
		// We wagen nog een poging binnen een uurtje
		// Toevoegen aan failqueue
		w.FailedQueue.Push(item, item.FailCount)
		w.crawler.speedLogger.NewFailedQueue++
	} else {
		w.crawler.speedLogger.LogUnavailable()
	}
}

func (w *Hostworker) GetNextRequest() *CrawlItem {
	f := w.FailedQueue.Pop()
	if f != nil {
		w.crawler.speedLogger.PoppedFromFailedQueue++
		return f
	}
	if !w.PriorityQueue.IsEmpty() {
		w.crawler.speedLogger.PoppedFromPriorityQueue++
		return w.PriorityQueue.Pop()
	}
	if !w.Queue.IsEmpty() {
		w.crawler.speedLogger.PoppedFromQueue++
		return w.Queue.Pop()
	}

	if w.LowPriorityQueue.IsEmpty() {
		return nil
	}

	w.crawler.speedLogger.PoppedFromLowPriorityQueue++
	return w.LowPriorityQueue.Pop()
}

func cleanURLPath(u *url.URL) (string, error) {
	normalized := purell.NormalizeURL(u,
		purell.FlagDecodeUnnecessaryEscapes|
			purell.FlagUppercaseEscapes|
			purell.FlagEncodeNecessaryEscapes|
			purell.FlagRemoveDefaultPort|
			purell.FlagRemoveEmptyQuerySeparator|
			purell.FlagRemoveFragment|
			purell.FlagRemoveEmptyPortSeparator|
			purell.FlagRemoveTrailingSlash)

	clean, err := url.ParseRequestURI(normalized)

	if err != nil {
		return "", err
	}

	return clean.EscapedPath(), nil
}

/**
 * Er werd een referentie gevonden naar een URL voor deze host
 * depth = nil als het van externe host komt. Anders is depth de diepte van het item waarvan de referentei afkomstig is
 */
func (w *Hostworker) NewReference(foundUrl *url.URL, depth *int, source *url.URL, sourceItem *CrawlItem) {
	uri, err := cleanURLPath(foundUrl)
	if err != nil {
		w.crawler.cfg.LogError(err)
		return
	}

	item, found := w.AlreadyVisited[uri]
	if !found {
		item = NewCrawlItem(foundUrl)
		w.AlreadyVisited[uri] = item
		w.crawler.speedLogger.NewURLsCount++
	} else {
		if item.IsUnavailable() {
			// Deze url is onbereikbaar, ofwel geen HTML bestand
			// dat weten we omdat we deze al eerder hebben gecrawled
			return
		}
	}

	now := time.Now()

	// Depth aanpassen
	if depth == nil {
		// Referentie vanaf een ander domein
		item.Depth = 0
		item.LastReference = &now
		item.LastReferenceURL = source

	} else {
		if !found || item.Depth >= *depth+1 {
			item.Depth = *depth + 1
			item.LastReference = &now
			item.LastReferenceURL = source
		}
	}

	if depth != nil && sourceItem != nil && item.Depth < maxRecrawlDepth && sourceItem.Depth > item.Depth && item.DownloadCount < sourceItem.DownloadCount && item.LastDownload != nil {
		// Het systeem is zo ontworpen dat een item in de priority queue enkel wordt gedownload nadat alle websites die er oorspronkelijk naar verwezen (met een lagere depth)
		// zijn gecrawled.
		// Hierdoor kunnen we verdwenen referenties detecteren in de priority queue. Verdwenen referenties in de gewone queue
		// zijn niet relevant aangezien die de gebruikte queue niet beïnvloeden.
		// Als een website met een hogere diepte verwijst naar een website met een lagere diepte, en als blijkt dat die website minder werd gedownload,
		// dan is er een referentie verloren.
		// Het kan ook gewoon zijn dat de pagina met de referentie bij de download onbereikbaar was en in de failedqueue staat, in dat geval
		// wordt die toch opnieuw gedownload  en als die nog bestaat zal de diepte terug worden aangepast naar een lagere diepte

		item.Depth = *depth + 1
		item.LastReference = &now
		item.LastReferenceURL = source
	}

	if item.Depth < maxRecrawlDepth && (item.Queue == w.Queue || item.Queue == w.LowPriorityQueue) {
		// Dit item staat nog in de gewone queue, maar heeft nu wel prioriteit
		// we verplaatsen het
		item.Remove()
		w.PriorityQueue.Push(item)

		w.crawler.speedLogger.SwitchesToPriority++

	} else if item.Queue == nil && (!found || (item.NeedsRecrawl() && item.LastReference == &now)) {
		// Enkel recrawl toelaten van referentie van lagere depth
		// Staat niet in een queue maar heeft wel een recrawl nodig

		if item.Depth < maxRecrawlDepth {
			w.PriorityQueue.Push(item)
			w.crawler.speedLogger.NewPriorityQueue++
		} else {
			if !found {
				w.Queue.Push(item)
				w.crawler.speedLogger.NewQueue++
			} else {
				w.LowPriorityQueue.Push(item)
				w.crawler.speedLogger.NewLowPriorityQueue++
			}
		}
	}
}
