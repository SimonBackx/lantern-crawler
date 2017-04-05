package crawler

import (
	//"fmt"
	"github.com/deckarep/golang-set"
	"math/rand"
	"net/http"
	"sync"
)

type DomainCrawler struct {
	Website        *Website
	ActiveRequests int
	Mutex          *sync.Mutex
	Queue          *CrawlQueue
	AlreadyVisited mapset.Set
	Active         bool
	Client         *http.Client

	// Aantal requests die nog voltooid moeten worden
	// voor hij overweegt om naar slaapstand te gaan
	// als er andere domeinen 'wachten'
	sleepAfter int
}

func (domainCrawler *DomainCrawler) String() string {
	return domainCrawler.Website.URL
}

func NewDomainCrawler(website *Website) *DomainCrawler {
	//fmt.Println("New domain", website.URL)

	domainCrawler := &DomainCrawler{Website: website, Mutex: &sync.Mutex{}, Queue: NewCrawlQueue(), AlreadyVisited: mapset.NewSet()}
	domainCrawler.sleep() // lock niet nodig in constructor
	return domainCrawler
}

func (domainCrawler *DomainCrawler) RequestStarted() {
}

func (domainCrawler *DomainCrawler) RequestFinished() {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	domainCrawler.ActiveRequests--
	domainCrawler.sleepAfter--
	if domainCrawler.sleepAfter <= 0 {
		// Ga in slaapmodus
		domainCrawler.sleep()
	}
}

func (domainCrawler *DomainCrawler) InMemory() bool {
	return domainCrawler.Queue != nil
}

func (domainCrawler *DomainCrawler) HasItemAvailable() *CrawlItem {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	if !domainCrawler.Active {
		// Geen items beschikbaar als we inactive zijn
		return nil
	}

	// Maximum 1 request toelaten tergelijk
	if domainCrawler.Queue.IsEmpty() || domainCrawler.ActiveRequests >= 1 {
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
	uri := item.URL.EscapedPath()

	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	if !domainCrawler.InMemory() {
		// Toevoegen aan een tijdelijke already visted lijst die
		// af en toe naar disk wordt geschreven
		return
	}

	if !domainCrawler.AlreadyVisited.Add(uri) {
		return
	}

	domainCrawler.Queue.Push(item)
}

func (domainCrawler *DomainCrawler) Wake(client *http.Client) {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()

	//fmt.Println("Domain", domainCrawler.Website.URL, "awoke")

	domainCrawler.Client = client
	domainCrawler.Active = true

	if !domainCrawler.InMemory() {
		// Lezen vanaf disk!
		domainCrawler.Load()
	}
}

func (domainCrawler *DomainCrawler) Sleep() {
	domainCrawler.Mutex.Lock()
	defer domainCrawler.Mutex.Unlock()
	domainCrawler.sleep()
}

/**
 * Ga naar slaapstand, enkel aanroepen binnen de mutex lock!
 * @param  {[type]} domainCrawler *DomainCrawler) Sleep( [description]
 * @return {[type]}               [description]
 */
func (domainCrawler *DomainCrawler) sleep() {
	// Mutex moet al locked zijn voor het aanroepen!
	//fmt.Println("Domain", domainCrawler.Website.URL, "went to sleep")

	domainCrawler.Active = false
	//domainCrawler.Client = nil // Niet doen -> moet nog gebruikt worden voor raadplegen

	// ResignAfter opnieuw invullen (minimum 1 request)
	domainCrawler.sleepAfter = rand.Intn(5) + 1
}

/**
 * Plaats vrijmaken door queue en already visted weg te schrijven naar disk
 * @param  {[type]} domainCrawler *DomainCrawler) Free( [description]
 * @return {[type]}               [description]
 */
func (domainCrawler *DomainCrawler) Free() {
	// Queue leegmaken en opslaan

	// Already visited leegmaken en opslaan
}

/**
 * Opgeslagen data lezen vanaf disk
 * @param  {[type]} domainCrawler *DomainCrawler) Free( [description]
 * @return {[type]}               [description]
 */
func (domainCrawler *DomainCrawler) Load() {
	// Queue leegmaken en opslaan

	// Already visited leegmaken en opslaan
}
