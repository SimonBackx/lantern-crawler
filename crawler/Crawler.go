package crawler

import (
	"bytes"
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/parser"
	"golang.org/x/net/proxy"
	"io"
	"net/http"
	"strings"
)

type Crawler struct {
	cfg       *config.CrawlerConfig
	transport *http.Transport
	client    *http.Client
	Queue     *CrawlItem
	QueueTail *CrawlItem
}

func NewCrawler(cfg *config.CrawlerConfig) *Crawler {
	torDialer, err := proxy.SOCKS5("tcp", cfg.TorProxyAddress, nil, proxy.Direct)

	if err != nil {
		cfg.LogError(err)
		return nil
	}

	transport := &http.Transport{Dial: torDialer.Dial}
	client := &http.Client{Transport: transport}

	return &Crawler{cfg: cfg, client: client, transport: transport}
}

func (crawler *Crawler) Pop() {
	if crawler.Queue == nil {
		return
	}

	crawler.Queue = crawler.Queue.Next
	if crawler.Queue == nil {
		crawler.QueueTail = nil
	}
}

func (crawler *Crawler) Push(item *CrawlItem) {
	if crawler.Queue == nil {
		crawler.Queue = item
		crawler.QueueTail = item
		return
	}
	crawler.QueueTail.Next = item
}

func (crawler *Crawler) Crawl() {
	// Todo: block for concurrency
	item := crawler.Queue

	if item == nil {
		crawler.cfg.LogError(&CrawlError{"Queue is empty"})
		return
	}
	crawler.Pop()

	var reader io.Reader
	if item.Body != nil {
		reader = strings.NewReader(*item.Body)
	}
	fmt.Println("Request:", item.URL.String())

	if request, err := http.NewRequest(item.Method, item.URL.String(), reader); err == nil {
		if response, err := crawler.client.Do(request); err == nil {
			crawler.ProcessResponse(item, response.Request, response)

		} else {
			crawler.cfg.LogError(err)
		}
	} else {
		crawler.cfg.LogError(err)
	}
}

func printHeader(header *http.Header) {
	buffer := bytes.NewBufferString("")
	header.Write(buffer)
	fmt.Println(buffer.String())
}

func (crawler *Crawler) ProcessResponse(item *CrawlItem, request *http.Request, response *http.Response) {
	fmt.Println("Request headers:")
	printHeader(&request.Header)
	fmt.Println("Response:", item.URL.String())
	fmt.Println("Status:", response.Status)
	//fmt.Println("Response headers:")
	//printHeader(&response.Header)

	// Doorgeven aan parser
	result, err := parser.Parse(response.Body, item.Website.GetParsers(request.URL))
	response.Body.Close()

	if err != nil {
		crawler.cfg.LogError(err)
		return
	}

	if result.Listing != nil {
		result.Listing.Print()
	} else {
		fmt.Println("No listing found")
	}

	if result.Links != nil {
		parser.PrintLinks(result.Links)

		for _, link := range result.Links {
			abs := request.URL.ResolveReference(&link.Href)
			web := GetWebsiteForDomain(abs.Hostname())
			newItem := NewCrawlItem(abs, web)
			crawler.Push(newItem)
		}
		// Alle links converteren naar juist
	} else {
		fmt.Println("No links found")
	}
	fmt.Println("")
}
