package crawler

import (
	//"bytes"
	//"fmt"
	"github.com/PuerkitoBio/purell"
	//"github.com/deckarep/golang-set"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	//"sync"
	"bufio"
	"bytes"
	"context"
	"os"
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
		crawler.cfg.LogInfo("Invalid file")
		return nil
	}
	//crawler.cfg.LogInfo("Reading " + string(host) + "...")

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
	duration := time.Second*30 - time.Since(*w.IntroductionPoints.First.LastDownload)

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

func (w *Hostworker) Recrawl() {
	w.crawler.cfg.LogInfo("Recrawl initiated for " + w.String())

	//w.crawler.cfg.LogInfo("Recrawl for host " + w.String@() + " initiated")
	// Recrawl nodig! -> Alles toevoegen aan de priority queue
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

	w.crawler.cfg.LogInfo("Goroutine for host " + w.String() + " started")

	w.Client = client

	// Snel horizontaal uitbreiden: neem laag getal
	w.sleepAfter = 200 //rand.Intn(60) + 1

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

			if w.sleepAfter <= 0 {
				// Meteen stoppen
				return
			}

			// Onderstaande kansverdeling moet nog minder uniform gemaakt worden
			time.Sleep(time.Millisecond*time.Duration(rand.Intn(4000)) + 1000)
		}
	}
}

func readFirstBytes(r io.Reader) ([]byte, error) {
	b := make([]byte, 512, 512)
	n, err := r.Read(b)
	if err == io.EOF {
		// done, maar snij onze byte slice bij om lege (niet ingelezen)
		// bytes te verwijderen
		return b[:n], nil
	}

	if err != nil {
		return b, err
	}

	// Er valt nog verder te lezen
	return b, nil
}

// readRemaining reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with the already read bytes
func readRemaining(r io.Reader, alreadyRead []byte) (reader io.Reader, err error) {
	buf := bytes.NewBuffer(alreadyRead)
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	_, err = buf.ReadFrom(r)

	return bytes.NewReader(buf.Bytes()), err
}

func (w *Hostworker) Request(item *CrawlItem) {
	var reader io.Reader
	/*if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}*/
	//idleChan := time.After(10 * time.Second)

	cx, cancel := context.WithCancel(context.Background())
	ref := &cancel

	if request, err := http.NewRequest("GET", item.URL.String(), reader); err == nil {
		request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0")
		request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		request.Header.Add("Accept_Language", "en-US,en;q=0.5")
		//request.Header.Add("Connection", "keep-alive")

		request.Close = true              // Connectie weggooien
		request = request.WithContext(cx) //request.WithContext(w.crawler.context)
		go func() {
			time.Sleep(3 * time.Second)
			c := ref
			if c != nil {
				w.crawler.speedLogger.Timeouts++
				(*c)()
			} else {
			}
		}()

		if response, err := w.Client.Do(request); err == nil {
			defer response.Body.Close()
			ref = nil

			if response.ContentLength > 1000000 {
				//w.crawler.cfg.LogInfo("Response: Content too long")
				// Too big
				// Eventueel op een ignore list zetten
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
				return
			}

			// Rest inlezen
			reader, err := readRemaining(response.Body, b)
			if err != nil {
				// Er ging iets mis
				w.crawler.cfg.LogError(err)
				w.RequestFailed(item)
				return
			}

			w.ProcessResponse(item, response, reader)
		} else {
			if response != nil && response.Body != nil {
				//w.crawler.cfg.LogError(err)
				response.Body.Close()
			}
			w.RequestFailed(item)
			//w.crawler.cfg.LogError(err)
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

		w.RequestFailed(item)
	}
}

func (w *Hostworker) ProcessResponse(item *CrawlItem, response *http.Response, reader io.Reader) {

	requestUrl := *response.Request.URL
	urlRef := &requestUrl

	/*buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	w.crawler.cfg.LogInfo(buf.String()) // Does a complete copy of the bytes in the buffer.*/

	// Doorgeven aan parser
	result, err := Parse(reader, w.crawler.Queries)

	if err != nil {
		//w.crawler.cfg.LogError(err)
		w.RequestFailed(item)
		/*if _, ok := err.(parser.ParseError); ok {
			w.crawler.cfg.LogError(err)
		} else {
			// not valid html or too long body
			w.crawler.cfg.LogError(err)
		}*/
		return
	}

	for _, query := range result.Queries {
		w.crawler.cfg.LogInfo("Found " + query.String() + " at " + w.String() + item.String())
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

			// Alle invalid characters verwijderen
			reg := regexp.MustCompile("[^\\^0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ,.\\-!/()=?`*;:_{}[]\\|~]+")
			u, err := url.Parse(reg.ReplaceAllString(u.String(), ""))
			if err != nil {
				break
			}

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

	// Resultaat doorgeven aan Crawler
	if len(workerResult.Links) > 0 {
		//w.crawler.cfg.LogInfo("Resultaat van worker verzonden " + w.String())
		w.crawler.WorkerResult <- workerResult
	}

	w.RequestFinished(item)
}

func (w *Hostworker) RequestStarted(item *CrawlItem) {
	// Ongeacht gelukt / mislukt (is noodzakelijk detectie verdwenen referenties)
	w.sleepAfter--
	item.DownloadCount++

	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request started. URL = %v", item.URL.String()))
	now := time.Now()
	item.LastDownloadStarted = &now
}

func (w *Hostworker) RequestFinished(item *CrawlItem) {
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request finished. URL = %v", item.URL.String()))
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

	w.crawler.speedLogger.Log()
	//w.crawler.cfg.LogInfo(fmt.Sprintf("Request finished. URL = %v", item.URL.String()))
}

func (w *Hostworker) RequestFailed(item *CrawlItem) {
	item.FailCount++
	if !item.IsUnavailable() {
		// We wagen nog een poging binnen een uurtje
		// Toevoegen aan failqueue
		w.FailedQueue.Push(item, item.FailCount)
		w.crawler.speedLogger.NewFailedQueue++
	} else {
		w.crawler.speedLogger.LogUnavailable()
	}
}

func (w *Hostworker) InMemory() bool {
	return w.Queue != nil
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

		//w.crawler.cfg.LogInfo("Referentie naar " + uri + " (" + w.Host + ") ging verloren")
		item.Depth = *depth + 1
		item.LastReference = &now
		item.LastReferenceURL = source
	}

	if item.Depth < maxRecrawlDepth && (item.Queue == w.Queue || item.Queue == w.LowPriorityQueue) {
		//w.crawler.cfg.LogInfo("Item " + uri + " (" + w.Host + ") had lage prioriteit, maar is nu hoge prioriteit")
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
