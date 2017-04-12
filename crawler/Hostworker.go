package crawler

import (
	//"bytes"
	//"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/SimonBackx/master-project/parser"
	//"github.com/deckarep/golang-set"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	//"regexp"
	"strings"
	//"sync"
	"bufio"
	"os"
	"time"
)

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
	//FailedQueue *CrawlQueue

	// Hierin staan alle items die aanwezig zijn in Queue of PriorityQueue
	// OF items die niet aanwezig zijn in die queue's, maar die nog niet gedownload mogen worden
	// omdat item.NeedRecrawl false geeft. We
	// gaan deze af en toe uitkuisen en items die NeedsRecrawl true geven verwijderen
	// zo geraken we bewust pagina's kwijt waar nergens nog naar wordt verwezen
	AlreadyVisited map[string]*CrawlItem

	// Lijst met alle url's met diepte = 0. Deze staan cronologisch gerangschikt
	// van laatste gedownload naar meest recent gedownload
	IntroductionPoints *CrawlQueue

	Running  bool // Of goroutine loopt
	Sleeping bool // Of deze worker in de sleeping queue aanwezig is

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
	file, err := os.Create("progress/" + w.Host + ".txt")
	if err != nil {
		w.crawler.cfg.LogError(err)
		return
	}
	defer func() {
		file.Close()
	}()

	writer := bufio.NewWriter(file)
	writer.WriteString(w.String())
	writer.WriteString("\nIntroduction points:\n")

	item := w.IntroductionPoints.First
	for item != nil {
		writer.WriteString("\n")
		writer.WriteString(item.SaveToString())
		item = item.Next
	}

	writer.WriteString("\nPriority:\n")
	item = w.PriorityQueue.First
	for item != nil {
		writer.WriteString("\n")
		writer.WriteString(item.SaveToString())
		item = item.Next
	}

	writer.WriteString("\nQueue:\n")
	item = w.Queue.First
	for item != nil {
		writer.WriteString("\n")
		writer.WriteString(item.SaveToString())
		item = item.Next
	}
	writer.WriteString("\nLow Priority:\n")
	item = w.LowPriorityQueue.First
	for item != nil {
		writer.WriteString("\n")
		writer.WriteString(item.SaveToString())
		item = item.Next
	}

	// todo: introduction points

	writer.Flush()
}

func NewHostworker(host string, crawler *Crawler) *Hostworker {
	w := &Hostworker{
		Host:               host,
		Queue:              NewCrawlQueue(),
		PriorityQueue:      NewCrawlQueue(),
		LowPriorityQueue:   NewCrawlQueue(),
		IntroductionPoints: NewCrawlQueue(),

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

func (w *Hostworker) Run(client *http.Client) {
	defer func() {
		// Aangeven dat deze goroutine afgelopen is
		w.crawler.waitGroup.Done()

		// Onze crawler terug wakker maken om eventueel een nieuwe request op te starten
		w.crawler.WorkerEnded <- w
	}()

	//w.crawler.cfg.LogInfo("Goroutine for host " + w.String() + " started")

	w.Client = client

	// Snel horizontaal uitbreiden: neem laag getal
	w.sleepAfter = rand.Intn(60) + 1

	if w.IntroductionPoints.First != nil && w.IntroductionPoints.First.NeedsRecrawl() {
		w.crawler.cfg.LogInfo("Recrawl for host " + w.String() + " initiated")
		// Recrawl nodig! -> Alles toevoegen aan de priority queue
		item := w.IntroductionPoints.First
		for item != nil {
			next := item.Next
			item.Remove()
			w.PriorityQueue.Push(item)

			item = next
		}
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

			w.RequestStarted(item)
			w.Request(item)
			w.RequestFinished(item)

			if w.sleepAfter <= 0 {
				// Meteen stoppen
				return
			}

			// Onderstaande kansverdeling moet nog minder uniform gemaakt worden
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(4000)))
		}
	}
}
func (w *Hostworker) Request(item *CrawlItem) {
	// Ongeacht gelukt / mislukt (is noodzakelijk detectie verdwenen referenties)
	item.DownloadCount++

	var reader io.Reader
	/*if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}*/

	if request, err := http.NewRequest("GET", item.URL.String(), reader); err == nil {
		request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0")
		request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		request.Header.Add("Accept_Language", "en-US,en;q=0.5")
		request.Header.Add("Connection", "keep-alive")

		request.Close = true // Connectie weggooien
		request = request.WithContext(w.crawler.context)

		if response, err := w.Client.Do(request); err == nil {
			w.ProcessResponse(item, response)
		} else {
			if response != nil && response.Body != nil {
				response.Body.Close()
			}

			//w.crawler.cfg.LogError(err)
			/*if urlErr, ok := err.(*url.Error); ok {
			      if netOpErr, ok := urlErr.Err.(*net.OpError); ok && netOpErr.Timeout() {
			          fmt.Println("Timeout: ", item.URL.String())
			      } else {
			      }
			  } else {
			      fmt.Println("Unknown error: ", err, item.URL.String())
			  }

			  fmt.Printf("%v, %T\n", err, err)*/
		}
	} else {
		//w.crawler.cfg.LogError(err)
	}
}

func (w *Hostworker) ProcessResponse(item *CrawlItem, response *http.Response) {
	defer response.Body.Close()

	requestUrl := *response.Request.URL
	urlRef := &requestUrl

	/*buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	w.crawler.cfg.LogInfo(buf.String()) // Does a complete copy of the bytes in the buffer.*/
	parsers := []parser.IParser{&parser.LinkParser{}}

	// Doorgeven aan parser
	result, err := parser.Parse(response.Body, parsers)

	if err != nil {
		/*if _, ok := err.(parser.ParseError); ok {
			w.crawler.cfg.LogError(err)
		} else {
			// not valid html or too long body
			w.crawler.cfg.LogError(err)
		}*/
		return
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

			/*if !strings.HasSuffix(u.Hostname(), ".onion") {
				break
			}

			if len(u.Host) != 22 {
				// todo: ondersteuning voor tor subdomains toevoegen!
				// Ongeldig -> verwijder alle ongeldige characters (tor browser doet dit ook)
				reg := regexp.MustCompile("[^a-zA-Z2-7.]+")
				u.Host = reg.ReplaceAllString(u.Host, "")
				if len(u.Host) != 22 {
					break
				}
			}*/

			normalized := purell.NormalizeURL(u,
				purell.FlagsSafe|purell.FlagRemoveFragment)

			u, err = url.ParseRequestURI(normalized)

			if err != nil {
				w.crawler.cfg.LogError(err)
				break
			}

			if u.Hostname() == w.Host {
				// Interne URL's meteen verwerken
				w.NewReference(u, &item.Depth, urlRef, item)
			} else {
				workerResult.Append(u)
			}
		}
	}

	now := time.Now()
	item.LastDownload = &now

	// Resultaat doorgeven aan Crawler
	if len(workerResult.Links) > 0 {
		w.crawler.WorkerResult <- workerResult
	}

	w.crawler.speedLogger.Log()
}

func (w *Hostworker) RequestStarted(item *CrawlItem) {
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request started. URL = %v", item.URL.String()))

}

func (w *Hostworker) RequestFinished(item *CrawlItem) {
	if item.Depth == 0 {
		// Introduction point toevoegen
		w.IntroductionPoints.Push(item)
	}
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request finished. URL = %v", item.URL.String()))
	w.sleepAfter--
}

func (w *Hostworker) InMemory() bool {
	return w.Queue != nil
}

func (w *Hostworker) GetNextRequest() *CrawlItem {
	if !w.PriorityQueue.IsEmpty() {
		return w.PriorityQueue.Pop()
	}
	if !w.Queue.IsEmpty() {
		return w.Queue.Pop()
	}

	return w.LowPriorityQueue.Pop()
}

/**
 * Er werd een referentie gevonden naar een URL voor deze host
 * depth = nil als het van externe host komt. Anders is depth de diepte van het item waarvan de referentei afkomstig is
 */
func (w *Hostworker) NewReference(foundUrl *url.URL, depth *int, source *url.URL, sourceItem *CrawlItem) {
	normalized := purell.NormalizeURL(foundUrl,
		purell.FlagDecodeUnnecessaryEscapes|
			purell.FlagUppercaseEscapes|
			purell.FlagEncodeNecessaryEscapes|
			purell.FlagRemoveDefaultPort|
			purell.FlagRemoveEmptyQuerySeparator|
			purell.FlagRemoveFragment|
			purell.FlagRemoveEmptyPortSeparator|
			purell.FlagRemoveTrailingSlash)

	u, err := url.ParseRequestURI(normalized)

	if err != nil {
		w.crawler.cfg.LogError(err)
		return
	}

	uri := u.EscapedPath()

	item, found := w.AlreadyVisited[uri]
	if !found {
		item = NewCrawlItem(foundUrl)
		w.AlreadyVisited[uri] = item
	}

	// Staat deze al in de queue?
	inQueue := false
	if item.Queue == w.Queue || item.Queue == w.PriorityQueue || item.Queue == w.LowPriorityQueue {
		inQueue = true
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

	if depth != nil && sourceItem != nil && sourceItem.Depth > item.Depth && item.DownloadCount < sourceItem.DownloadCount && item.LastDownload != nil {
		// Het kan zijn dat dit item niet meer gerefereerd werd door een pagina met lagere depth,
		// sinds zijn laatste download
		// In dat geval is de referentie verdwenen, aangezien alle pagina's met een lagere depth gegarandeerd
		// eerder werden gerecrawld
		// en de referentie dus opnieuw had moeten worden gevonden
		//
		// Dit mag enkel effect hebben als het item in de priority queue terecht hoort, maar nu niet meer
		//if (*item.LastReference).Sub(*item.LastDownload) < 0 {
		// Er werd geen nieuwe referentie gevonden na de laatste download
		// lastReference is dus kleiner dan lastDownload
		w.crawler.cfg.LogInfo("Referentie naar " + uri + " (" + w.Host + ") ging verloren")
		item.Depth = *depth + 1
		item.LastReference = &now
		item.LastReferenceURL = source
		//}
	}

	if inQueue {
		if item.Depth < maxRecrawlDepth && (item.Queue == w.Queue || item.Queue == w.LowPriorityQueue) {
			w.crawler.cfg.LogInfo("Item " + uri + " (" + w.Host + ") had lage prioriteit, maar is nu hoge prioriteit")
			// Dit item staat nog in de gewone queue, maar heeft nu wel prioriteit
			// we verplaatsen het
			item.Remove()
			w.PriorityQueue.Push(item)
		}
	} else if !found || (item.NeedsRecrawl() && item.LastReference == &now) {
		if item.Queue != nil {
			// Kan zijn dat dit item in de introductionpoints queue staat
			// Recrawl zal dan de volgende goroutine gestart worden
			return
		}
		// Enkel recrawl toelaten van referentie van lagere depth

		// Staat niet in een queue maar heeft wel een recrawl nodig
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
}

/**
 * Plaats vrijmaken door queue en already visted weg te schrijven naar disk
 * @param  {[type]} w *Hostworker) Free( [description]
 * @return {[type]}               [description]
 */
func (w *Hostworker) Free() {
	// Queue leegmaken en opslaan

	// Already visited leegmaken en opslaan
}

/**
 * Opgeslagen data lezen vanaf disk
 * @param  {[type]} w *Hostworker) Free( [description]
 * @return {[type]}               [description]
 */
func (w *Hostworker) Load() {
	// Queue leegmaken en opslaan

	// Already visited leegmaken en opslaan
}
