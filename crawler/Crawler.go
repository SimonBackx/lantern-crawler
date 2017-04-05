package crawler

import (
	"bytes"
	"context"
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/parser"
	//"golang.org/x/net/proxy"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	//"time"
)

type Crawler struct {
	cfg           *config.CrawlerConfig
	distributor   ClientDistributor
	context       context.Context
	cancelContext context.CancelFunc

	// Map met alle URL -> DomainCrawlers (voor snel opzoeken)
	DomainCrawlers      map[string]*DomainCrawler
	DomainCrawlersMutex *sync.Mutex

	// DomainCrawlers die klaar staan om wakker gemaakt te worden maar geen requests uitvoeren
	SleepingCrawlers *DomainList

	// DomainCrawlers die wakker zijn maar niet met een request bezig zijn
	AvailableCrawlers *DomainList

	ResumeChannel chan bool

	// Waitgroup die we gebruiken als we op alle requests willen wachten
	waitGroup   sync.WaitGroup
	speedLogger *SpeedLogger
}

func NewCrawler(cfg *config.CrawlerConfig) *Crawler {
	/*if cfg.TorProxyAddress != nil {
		torDialer, err := proxy.SOCKS5("tcp", *cfg.TorProxyAddress, nil, proxy.Direct)

		if err != nil {
			cfg.LogError(err)
			return nil
		}
		transport = &http.Transport{
			Dial: torDialer.Dial,
		}
	} else {
		transport = &http.Transport{}
	}

	client := &http.Client{Transport: transport, Timeout: time.Second * 10}*/
	ctx, cancelCtx := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	return &Crawler{cfg: cfg,
		distributor:         NewClearnetDistributor(),
		context:             ctx,
		cancelContext:       cancelCtx,
		waitGroup:           wg,
		DomainCrawlers:      make(map[string]*DomainCrawler),
		SleepingCrawlers:    NewDomainList(),
		AvailableCrawlers:   NewDomainList(),
		ResumeChannel:       make(chan bool, 1),
		speedLogger:         NewSpeedLogger(),
		DomainCrawlersMutex: &sync.Mutex{},
	}
}

func (crawler *Crawler) AddDomain(domainCrawler *DomainCrawler) {
	crawler.DomainCrawlers[domainCrawler.Website.URL] = domainCrawler
}

func (crawler *Crawler) Wake() {
	select {
	case crawler.ResumeChannel <- true:
		//fmt.Println("Waked!")
	default:
		//fmt.Println("Not waking, already awake.")
	}
}

func (crawler *Crawler) Start(signal chan int) {
	crawler.cfg.LogInfo("Crawler started")

	for {

		// Hier moet nog locking toegevoegd worden
		// Controleren of we eventueel slapende domeinen kunnen wakker maken

		crawler.SleepingCrawlers.Lock()
		for i := crawler.SleepingCrawlers.First; i != nil; i = i.Next {
			domainCrawler := i.DomainCrawler
			client := crawler.distributor.GetClient()
			if client != nil {
				domainCrawler.Wake(client)

				length := crawler.SleepingCrawlers.Length()
				i.Remove()
				if length-1 != crawler.SleepingCrawlers.Length() {
					crawler.cfg.Log("ERROR", "Remove sleeping crawlers failed")
				}

				crawler.AvailableCrawlers.Append(domainCrawler)
			} else {
				break
			}
		}
		crawler.SleepingCrawlers.Unlock()

		// Hier moet nog locking toegevoegd worden
		crawler.AvailableCrawlers.Lock()
		for i := crawler.AvailableCrawlers.First; i != nil; i = i.Next {
			domainCrawler := i.DomainCrawler

			// Kunnen we nog een request uitvoeren?
			// ActiveRequests wordt enkel single threated vanuit deze goroutine aangeroepen.

			// Is dit domein beschikbaar?
			if item := domainCrawler.HasItemAvailable(); item != nil {
				// Mogelijk maken om op onze goroutines te wachten
				crawler.waitGroup.Add(1)

				// domein uit actieve lijst halen (zodat we deze niet nodeloos overlopen)
				length := crawler.AvailableCrawlers.Length()
				i.Remove()
				if length-1 != crawler.AvailableCrawlers.Length() {
					crawler.cfg.Log("ERROR", "Remove available failed")
				}

				go crawler.Crawl(item, domainCrawler)
			} else {
				domainCrawler.Sleep()

				// Kanaal vrij geven
				crawler.distributor.FreeClient(domainCrawler.Client)
				domainCrawler.Client = nil

				length := crawler.AvailableCrawlers.Length()
				i.Remove()
				if length-1 != crawler.AvailableCrawlers.Length() {
					crawler.cfg.Log("ERROR", "Remove available (empty) failed")
				}

				crawler.SleepingCrawlers.Append(domainCrawler)

				// Meteen verder gaan, want anders kunnen we naar deadlock gaan
				crawler.Wake()
				break
			}
		}
		crawler.AvailableCrawlers.Unlock()

		//fmt.Println("Loop end -  blocking")
		// We hebben alles gestart wat we konden starten.
		// Nu wachten we tot er iets aan de situatie veranderd is
		// Wacht tot iemand crawler.Wake() aanroept
		<-crawler.ResumeChannel

		// Ontvangen we een quit signaal?
		select {
		case code := <-signal:
			if code == 1 {
				crawler.cfg.LogInfo("Stopping crawler...")
				crawler.cancelContext()
				// Wacht tot de context is beïndigd
				<-crawler.context.Done()

				// Wachten tot alle goroutines afgelopen zijn die requests verwerken
				crawler.waitGroup.Wait()

				/*for _, domainCrawler := range crawler.DomainCrawlers {
					crawler.cfg.LogInfo(fmt.Sprintf("Queue remaining for %v:", domainCrawler.Website.URL))
					domainCrawler.Queue.PrintQueue()
					fmt.Println()
				}*/

				crawler.cfg.LogInfo("Sleeping domains:")
				crawler.SleepingCrawlers.Print()
				fmt.Println()

				crawler.cfg.LogInfo("Available domains:")
				crawler.AvailableCrawlers.Print()
				fmt.Println()

				crawler.cfg.LogInfo("The crawler has stopped")
				return
			}
		default:
		}
	}
}

func (crawler *Crawler) Crawl(item *CrawlItem, domainCrawler *DomainCrawler) {
	defer func() {
		crawler.speedLogger.Log()

		domainCrawler.RequestFinished()
		if !domainCrawler.Active {
			crawler.SleepingCrawlers.Append(domainCrawler)

			// Kanaal vrij geven
			crawler.distributor.FreeClient(domainCrawler.Client)
			domainCrawler.Client = nil

		} else {
			// Wachten voor we volgende request starten
			//time.Sleep(2 * time.Second)

			// Terug toevoegen aan active crawlers
			crawler.AvailableCrawlers.Append(domainCrawler)
		}

		// Aangeven dat deze goroutine afgelopen is
		crawler.waitGroup.Done()

		// Onze crawler terug wakker maken om eventueel een nieuwe request op te starten
		crawler.Wake()
	}()

	domainCrawler.RequestStarted()

	var reader io.Reader
	if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}
	//crawler.cfg.LogInfo(fmt.Sprintf("Request started. URL = %v", item.URL.String()))

	if request, err := http.NewRequest(item.Method, item.URL.String(), reader); err == nil {
		request.Header.Add("Accept", "text/html")
		request = request.WithContext(crawler.context)

		if response, err := domainCrawler.Client.Do(request); err == nil {
			crawler.ProcessResponse(item, domainCrawler, response.Request, response)
		} else {
			/*if urlErr, ok := err.(*url.Error); ok {
				if netOpErr, ok := urlErr.Err.(*net.OpError); ok && netOpErr.Timeout() {
					fmt.Println("Timeout: ", item.URL.String())
				} else {
				}
			} else {
				fmt.Println("Unknown error: ", err, item.URL.String())
			}

			fmt.Printf("%v, %T\n", err, err)*/
			crawler.cfg.LogError(err)
		}
	} else {
		crawler.cfg.LogError(err)
	}
}

func PrintHeader(header *http.Header) {
	buffer := bytes.NewBufferString("")
	header.Write(buffer)
	fmt.Println(buffer.String())
}

func (crawler *Crawler) ProcessResponse(item *CrawlItem, domainCrawler *DomainCrawler, request *http.Request, response *http.Response) {
	defer response.Body.Close()

	/*fmt.Println("Request headers:")
	PrintHeader(&request.Header)*/
	//crawler.cfg.LogInfo(fmt.Sprintf("Response received. URL = %v", item.URL.String()))

	//fmt.Println("Status:", response.Status)
	//fmt.Println("Response headers:")
	//printHeader(&response.Header)
	url := *request.URL
	urlRef := &url

	// Doorgeven aan parser
	result, err := parser.Parse(response.Body, domainCrawler.Website.GetParsers(urlRef))

	if err != nil {
		// Deze error is voornamelijk
		if _, ok := err.(parser.ParseError); ok {
			crawler.cfg.LogError(err)
		} else {
			//fmt.Println("Error: not valid html or too long body")
			crawler.cfg.LogError(err)
		}

		return
	}

	if result.Listing != nil {
		result.Listing.Print()
	} else {
		//fmt.Println("No listing found")
	}

	if result.Links != nil {
		for _, link := range result.Links {
			// Convert links to absolute url
			abs := urlRef.ResolveReference(&link.Href)
			crawler.ProcessUrl(abs)
		}
	} else {
		//fmt.Println("No links found")
	}
	//fmt.Println("")
}

func (crawler *Crawler) ProcessUrl(url *url.URL) {
	if url == nil || crawler == nil {
		return
	}
	if !url.IsAbs() {
		//crawler.cfg.LogInfo("URL is not absolute: " + url.RequestURI())
		return
	}
	// Is deze URL één van onze domain crawlers?

	domain := url.Hostname()

	if len(domain) < 3 {
		//crawler.cfg.LogInfo("URL has no domain: " + url.RequestURI())
		return
	}
	crawler.DomainCrawlersMutex.Lock()
	defer crawler.DomainCrawlersMutex.Unlock()

	domainCrawler := crawler.DomainCrawlers[domain]

	if domainCrawler == nil {
		website := &Website{
			URL: domain,
		}
		domainCrawler = NewDomainCrawler(website)
		crawler.DomainCrawlers[domain] = domainCrawler
		crawler.SleepingCrawlers.Append(domainCrawler)
	}

	domainCrawler.AddItem(NewCrawlItem(url))
}
