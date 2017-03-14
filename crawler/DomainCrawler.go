package crawler

import (
	"fmt"
	"sync"
)

type DomainCrawler struct {
	Website        *Website
	ActiveRequests int
	Mutex          *sync.Mutex
	Queue          *CrawlQueue
}

func NewDomainCrawler(website *Website) *DomainCrawler {
	return &DomainCrawler{Website: website, Mutex: &sync.Mutex{}, Queue: NewCrawlQueue()}
}

func (domainCrawler *DomainCrawler) DecreaseActiveRequests() {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	domainCrawler.ActiveRequests--
	fmt.Println("Active Requests:", domainCrawler.ActiveRequests)
}

func (domainCrawler *DomainCrawler) HasItemAvailable() *CrawlItem {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	if domainCrawler.Queue.IsEmpty() || domainCrawler.ActiveRequests >= 2 {
		return nil
	}

	domainCrawler.ActiveRequests++
	item := domainCrawler.Queue.Pop()

	if item == nil {
		panic("Popped Queue is nil after checking empty... Are you using DomainCrawler.Queue outside the mutex?")
	}

	return item
}
