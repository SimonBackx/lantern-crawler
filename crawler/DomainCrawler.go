package crawler

import (
	"github.com/deckarep/golang-set"
	"sync"
)

type DomainCrawler struct {
	Website        *Website
	ActiveRequests int
	Mutex          *sync.Mutex
	Queue          *CrawlQueue
	AlreadyVisited mapset.Set
}

func NewDomainCrawler(website *Website) *DomainCrawler {
	return &DomainCrawler{Website: website, Mutex: &sync.Mutex{}, Queue: NewCrawlQueue(), AlreadyVisited: mapset.NewSet()}
}

func (domainCrawler *DomainCrawler) DecreaseActiveRequests() {
	domainCrawler.Mutex.Lock()
	domainCrawler.ActiveRequests--
	domainCrawler.Mutex.Unlock()
}

func (domainCrawler *DomainCrawler) HasItemAvailable() *CrawlItem {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	if domainCrawler.Queue.IsEmpty() || domainCrawler.ActiveRequests >= domainCrawler.Website.MaxRequests {
		return nil
	}

	domainCrawler.ActiveRequests++
	item := domainCrawler.Queue.Pop()

	if item == nil {
		panic("Popped Queue is nil after checking empty... Are you using DomainCrawler.Queue outside the mutex?")
	}

	return item
}

func (domainCrawler *DomainCrawler) AddItem(item *CrawlItem) {
	uri := item.URL.RequestURI()

	// Beoordeel dit bestand: we willen enkel text bestanden
	//

	if !domainCrawler.AlreadyVisited.Add(uri) {
		return
	}

	domainCrawler.Mutex.Lock()
	domainCrawler.Queue.Push(item)
	domainCrawler.Mutex.Unlock()
}
