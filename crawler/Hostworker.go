package crawler

import (
	//"bytes"
	"fmt"
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
	"time"
)

type Hostworker struct {
	Website *Website

	// De queue bevat een lijst met items die nog niet gecrawled werden en
	// één voor één zullen worden gedownload
	Queue *CrawlQueue

	// RecrawlQueue bevat items die we succesvol hebben kunnen crawlen en die na
	// een bepaalde periode opnieuw moeten worden gecrawled
	RecrawlQueue *RecrawlQueue

	// Lijst met items die te veel na elkaar mislukt zijn
	// Ze staan chonologisch gerangschikt. Degene die het langst geleden mislukt zijn,
	// staan vooraan in de wachtrij. Regelmatig controleren we of het eerste item ouder is dan 12 uur.
	// als dat zo is, halen we deze uit de lijst en verwijderen we deze uit AlreadyVisited om het opnieuw een kans te geven
	// om gecrawled te worden.
	//FailedQueue *CrawlQueue

	// Hierin staan alle URL's van items waarvan hun URL al ontdekt werd
	// en dus aanwezig zijn in Queue, RecrawlQueue of FailedQueue
	AlreadyVisited map[string]*CrawlItem //mapset.Set

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
	return w.Website.URL
}

func NewHostworker(website *Website, crawler *Crawler) *Hostworker {
	w := &Hostworker{
		Website:        website,
		Queue:          NewCrawlQueue(),
		RecrawlQueue:   NewRecrawlQueue(),
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

func (w *Hostworker) AddQueue(q *CrawlQueue) {
	// Eerst nog overlopen op already visited, we kunnen dus niet rechtstreeks merge gebruiken
	item := q.First
	for item != nil {
		w.AddItem(item, nil)
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
	w.sleepAfter = rand.Intn(10) + 2

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
			time.Sleep(time.Second * time.Duration(rand.Intn(5)+1))
		}
	}
}
func (w *Hostworker) Request(item *CrawlItem) {
	var reader io.Reader
	if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}

	if request, err := http.NewRequest(item.Method, item.URL.String(), reader); err == nil {
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

			w.crawler.cfg.LogError(err)
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
		w.crawler.cfg.LogError(err)
	}
}

func (w *Hostworker) ProcessResponse(item *CrawlItem, response *http.Response) {
	defer response.Body.Close()

	requestUrl := *response.Request.URL
	urlRef := &requestUrl

	/*buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	w.crawler.cfg.LogInfo(buf.String()) // Does a complete copy of the bytes in the buffer.*/

	// Doorgeven aan parser
	result, err := parser.Parse(response.Body, w.Website.GetParsers(urlRef))

	if err != nil {
		if _, ok := err.(parser.ParseError); ok {
			w.crawler.cfg.LogError(err)
		} else {
			// not valid html or too long body
			w.crawler.cfg.LogError(err)
		}
		return
	}

	workerResult := NewWorkerResult()

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

			if u.Hostname() == w.Website.URL {
				// Interne URL's meteen verwerken
				w.AddItem(NewCrawlItem(u), &item.Depth)
			} else {
				workerResult.Append(u)
			}
		}
	}

	if item.LastDownload == nil {
		w.RecrawlQueue.Push(item)
	}
	now := time.Now()
	item.LastDownload = &now

	// Resultaat doorgeven aan Crawler
	if len(workerResult.Links) > 0 {
		w.crawler.WorkerResult <- workerResult
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		w.crawler.speedLogger.Log()
	}
}

func (w *Hostworker) RequestStarted(item *CrawlItem) {
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request started. URL = %v", item.URL.String()))

}

func (w *Hostworker) RequestFinished(item *CrawlItem) {
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request finished. URL = %v", item.URL.String()))
	w.sleepAfter--
}

func (w *Hostworker) InMemory() bool {
	return w.Queue != nil
}

func (w *Hostworker) GetNextRequest() *CrawlItem {
	if w.Queue.IsEmpty() {
		return nil
	}

	item := w.Queue.Pop()
	if item == nil {
		panic("Popped Queue is nil after checking empty... Are you using Hostworker.Queue outside the mutex?")
	}

	return item
}

/**
 * depth = nil als het van externe host komt. Anders is depth de diepte van het item waarvan de referentei afkomstig is
 */
func (w *Hostworker) AddItem(item *CrawlItem, depth *int) {
	normalized := purell.NormalizeURL(item.URL,
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

	if !w.InMemory() {
		// Toevoegen aan een tijdelijke already visted lijst die
		// af en toe naar disk wordt geschreven
		return
	}

	existingItem, found := w.AlreadyVisited[uri]
	if !found {
		existingItem = item

		// Nieuwe url!
		w.AlreadyVisited[uri] = existingItem

		// Toevoegen aan download queue
		w.Queue.Push(existingItem)
	}

	// Referentie tijd aanpassen doen we enkel als het een referentie is
	// van een lagere diepte of externe host

	now := time.Now()

	if depth == nil {
		existingItem.LastReference = &now

		// Van extern domein: depth op 0 zetten
		if existingItem.Depth != 0 {
			existingItem.Depth = 0

			w.crawler.cfg.LogInfo(fmt.Sprintf("Moved url %s (%s) to depth 0", uri, w))

			// Verplaatsen in recrawlqueue indien al eens gecrawled
			w.RecrawlQueue.DepthUpdated(existingItem)
		}
	} else {
		// Van dezelfde host (interne site link)
		if !found || existingItem.Depth > *depth {
			existingItem.LastReference = &now

			// Als er een referentie is waardoor de diepte van deze pagina daalt...
			if existingItem.Depth != *depth+1 {
				// Aanpassen indien dat nog niet gedaan werd
				existingItem.Depth = *depth + 1
				w.crawler.cfg.LogInfo(fmt.Sprintf("Moved url %s (%s) to depth %v", uri, w, *depth+1))

				// Verplaatsen in recrawlqueue indien al eens gecrawled
				w.RecrawlQueue.DepthUpdated(existingItem)
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
