package crawler

import (
	"bytes"
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/parser"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Crawler struct {
	cfg            *config.CrawlerConfig
	transport      *http.Transport
	client         *http.Client
	DomainCrawlers map[string]*DomainCrawler

	ResumeChannel chan bool
}

func NewCrawler(cfg *config.CrawlerConfig) *Crawler {
	torDialer, err := proxy.SOCKS5("tcp", cfg.TorProxyAddress, nil, proxy.Direct)

	if err != nil {
		cfg.LogError(err)
		return nil
	}

	transport := &http.Transport{Dial: torDialer.Dial}
	client := &http.Client{Transport: transport, Timeout: time.Second * 5}

	return &Crawler{cfg: cfg, client: client, transport: transport, DomainCrawlers: make(map[string]*DomainCrawler), ResumeChannel: make(chan bool, 1)}
}

func (crawler *Crawler) AddDomain(domainCrawler *DomainCrawler) {
	crawler.DomainCrawlers[domainCrawler.Website.URL] = domainCrawler
}

func (crawler *Crawler) Wake() {
	select {
	case crawler.ResumeChannel <- true:
		fmt.Println("Waked!")
	default:
		fmt.Println("Not waking, already awake.")
	}
}

func (crawler *Crawler) Start() {
	for {

		fmt.Println("Loop start - unblocked")
		for _, domainCrawler := range crawler.DomainCrawlers {
			// Kunnen we nog een request uitvoeren?
			// ActiveRequests wordt enkel single threated vanuit deze goroutine aangeroepen.

			for {
				if item := domainCrawler.HasItemAvailable(); item != nil {
					go crawler.Crawl(item, domainCrawler)
				} else {
					break
				}
			}
		}

		fmt.Println("Loop end -  blocking")
		// We hebben alles gestart wat we konden starten.
		// Nu wachten we tot er iets aan de situatie veranderd is
		<-crawler.ResumeChannel
	}
}

func (crawler *Crawler) Crawl(item *CrawlItem, domainCrawler *DomainCrawler) {
	defer func() {
		domainCrawler.DecreaseActiveRequests()
		//time.Sleep(2 * time.Second)
		crawler.Wake()
	}()

	var reader io.Reader
	if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}
	fmt.Println("Request:", item.URL.String())

	if request, err := http.NewRequest(item.Method, item.URL.String(), reader); err == nil {
		request.Header.Add("Accept", "text/html")
		if response, err := crawler.client.Do(request); err == nil {
			crawler.ProcessResponse(item, domainCrawler, response.Request, response)
		} else {
			if urlErr, ok := err.(*url.Error); ok {
				if netOpErr, ok := urlErr.Err.(*net.OpError); ok && netOpErr.Timeout() {
					fmt.Println("Timeout: ", item.URL.String())
				} else {
				}
			} else {
				fmt.Println("Unknown error: ", err, item.URL.String())
			}

			fmt.Printf("%v, %T\n", err, err)
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
	fmt.Println("Response:", item.URL.String())
	//fmt.Println("Status:", response.Status)
	//fmt.Println("Response headers:")
	//printHeader(&response.Header)
	url := *request.URL
	urlRef := &url

	// Doorgeven aan parser
	result, err := parser.Parse(response.Body, domainCrawler.Website.GetParsers(urlRef))

	if err != nil {
		crawler.cfg.LogError(err)
		return
	}

	if result.Listing != nil {
		//result.Listing.Print()
	} else {
		//fmt.Println("No listing found")
	}

	if result.Links != nil {
		for _, link := range result.Links {
			if link == nil {
				panic("link is nil")
			}
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
	// Is deze URL één van onze domain crawlers?

	domain := url.Hostname()
	domainCrawler := crawler.DomainCrawlers[domain]

	if domainCrawler != nil {
		domainCrawler.AddItem(NewCrawlItem(url))
	} else {
		// todo: deze url ergens heen sturen voor latere verwerking
	}
}
