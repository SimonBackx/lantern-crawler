package crawler

import (
	"bytes"
	"context"
	"fmt"
	"github.com/SimonBackx/master-project/config"
	//"golang.org/x/net/proxy"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Crawler struct {
	cfg           *config.CrawlerConfig
	distributor   ClientDistributor
	context       context.Context
	cancelContext context.CancelFunc

	// Map met alle URL -> DomainCrawlers (voor snel opzoeken)
	Workers map[string]*Hostworker

	// DomainCrawlers die klaar staan om wakker gemaakt te worden maar geen requests uitvoeren
	SleepingCrawlers *WorkerList

	// Lijst met workers gerangschikt op basis van wanneer ze
	// opnieuw gecrawld moeten worden. De workers die als eerste een recrawl
	// moeten beginnen staan voorraan.
	RecrawlList *WorkerList

	// Kanaal waarop een bericht zal worden verstuurd als het tijd is
	// om één of meerdere items van de RecrawlList te halen
	RecrawlTimer <-chan time.Time

	WorkerEnded        chan *Hostworker
	WorkerResult       chan *WorkerResult
	WorkerIntroduction chan *Hostworker

	// Waitgroup die we gebruiken als we op alle requests willen wachten
	waitGroup   sync.WaitGroup
	speedLogger *SpeedLogger
	Stop        chan struct{}

	Started bool
	Signal  chan int
	Queries []*Query
}

func NewCrawler(cfg *config.CrawlerConfig) *Crawler {
	ctx, cancelCtx := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	crawler := &Crawler{cfg: cfg,
		distributor:        NewClearnetDistributor(),
		context:            ctx,
		cancelContext:      cancelCtx,
		waitGroup:          wg,
		Workers:            make(map[string]*Hostworker),
		SleepingCrawlers:   NewWorkerList(),
		RecrawlList:        NewWorkerList(),
		WorkerEnded:        make(chan *Hostworker, 10),
		WorkerResult:       make(chan *WorkerResult, 50),
		WorkerIntroduction: make(chan *Hostworker, 50),
		speedLogger:        NewSpeedLogger(),
		Stop:               make(chan struct{}, 1),
		RecrawlTimer:       make(<-chan time.Time, 1),
		Queries:            make([]*Query, 0),
	}
	crawler.speedLogger.Crawler = crawler

	// Read from files
	files, _ := ioutil.ReadDir("./progress")
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			// Hidden files negeren
			continue
		}

		file, err := os.Open("./progress/" + f.Name())
		if err != nil {
			cfg.LogError(err)
			continue
		}

		worker := NewHostWorkerFromFile(file, crawler)
		file.Close()

		if worker != nil {
			crawler.Workers[worker.Host] = worker

			if worker.WantsToGetUp() {
				worker.Sleeping = true
				crawler.SleepingCrawlers.Push(worker)
			}
		}
	}

	return crawler
}

func (crawler *Crawler) ProcessUrl(url *url.URL, source *url.URL) {
	host := url.Hostname()
	worker := crawler.Workers[host]

	if worker == nil {
		if len(crawler.Workers) > 1 {
			// Debug: maximum aantal domains
			return
		}

		worker = NewHostworker(host, crawler)
		crawler.Workers[host] = worker
	}

	// Crawler queue pushen
	if worker.Running {
		// Pushen d.m.v. channel om concurrency problemen te vermijden
		// todo: stack maken van url's ipv crawlitems
		item := NewCrawlItem(url)
		item.LastReferenceURL = source
		worker.NewItems.stack(item)
	} else {
		// Geen concurrency problemen mogelijk
		// NewReference kan url ook weggooien als die al gecrawled is
		// Depth = nil, want dit is altijd van een externe host
		worker.NewReference(url, nil, source, nil)

		if !worker.Sleeping && worker.WantsToGetUp() {
			// Dit domein had geen items, maar nu wel
			worker.Sleeping = true
			crawler.SleepingCrawlers.Push(worker)
		}
	}
}

func (crawler *Crawler) AddQuery(q *Query) {
	crawler.Queries = append(crawler.Queries, q)
}

func (crawler *Crawler) WakeSleepingWorkers() {
	for crawler.SleepingCrawlers.First != nil {
		worker := crawler.SleepingCrawlers.First.Worker

		if !worker.WantsToGetUp() {
			crawler.Panic("Worker " + worker.String() + " heeft lege queue maar staat in sleeping crawlers")
			return
		}

		if worker.Running {
			crawler.Panic("Worker " + worker.String() + " is al opgestart maar staat in sleeping crawlers")
			return
		}

		if !worker.Sleeping {
			crawler.Panic("Worker " + worker.String() + " .Sleeping = false maar staat in sleepingCrawlers")
			return
		}

		client := crawler.distributor.GetClient()
		if client == nil {
			// Geen client meer beschikbaar
			break
		}

		// Verwijderen uit queue
		crawler.SleepingCrawlers.Pop()

		// Goroutine starten
		worker.Running = true
		worker.Sleeping = false
		crawler.waitGroup.Add(1)
		go worker.Run(client)
	}
}

func (crawler *Crawler) SetRecrawlFirst(worker *Hostworker) {
	duration := worker.GetRecrawlDuration()
	// Minimale wachttijd
	if duration < time.Second*5 {
		duration = time.Second * 5
	}

	crawler.RecrawlTimer = time.After(duration)
}

func (crawler *Crawler) AddRecrawlList(worker *Hostworker) {
	crawler.cfg.LogInfo("Added to recrawl list: " + worker.String())

	if crawler.RecrawlList.IsEmpty() {
		crawler.SetRecrawlFirst(worker)
	}

	crawler.RecrawlList.Push(worker)
	worker.InRecrawlList = true
}

func (crawler *Crawler) CheckRecrawlList() {
	crawler.cfg.LogInfo("Check recrawl list")

	for crawler.RecrawlList.First != nil {
		if crawler.RecrawlList.First.Worker.GetRecrawlDuration() > 0 {
			// Lijst is nog niet leeg, maar is nog niet beschikbaar
			crawler.SetRecrawlFirst(crawler.RecrawlList.First.Worker)
			break
		}
		// Deze worker moet hercrawled worden
		worker := crawler.RecrawlList.Pop()
		worker.InRecrawlList = false

		if worker.Running {
			// Recrawl starten als worker eindigt
			worker.RecrawlOnFinish = true
		} else {
			// Meteen live toevoegen
			worker.Recrawl()

			if !worker.Sleeping && worker.WantsToGetUp() {
				// Deze worker had geen items, maar nu wel
				worker.Sleeping = true
				crawler.SleepingCrawlers.Push(worker)
			}
		}
	}
}

func (crawler *Crawler) Panic(str string) {
	crawler.cfg.Log("PANIC", str)
	select {
	case crawler.Signal <- 1:
		break
	default:
		break
	}
}

func (crawler *Crawler) Quit() {
	crawler.cfg.LogInfo("Stopping crawler...")
	crawler.speedLogger.Ticker.Stop()

	close(crawler.Stop)
	crawler.cancelContext()

	// Wacht tot de context is beïndigd
	<-crawler.context.Done()

	// Wachten tot alle goroutines afgelopen zijn die requests verwerken
	crawler.waitGroup.Wait()

	crawler.cfg.LogInfo("Saving progress...")
	// Tijdelijk uitgeschakeld voor debugging
	/*for _, worker := range crawler.Workers {
		worker.SaveToFile()
	}*/

	/*for _, worker := range crawler.Workers {
		fmt.Printf("Remaining queue for %s\n", worker)
		worker.RecrawlQueue.PrintQueue()
	}*/

	/*for _, domainCrawler := range crawler.DomainCrawlers {
		crawler.cfg.LogInfo(fmt.Sprintf("Queue remaining for %v:", domainCrawler.Website.URL))
		domainCrawler.Queue.PrintQueue()
		fmt.Println()
	}*/

	/*crawler.cfg.LogInfo("Sleeping domains:")
	crawler.SleepingCrawlers.Print()
	fmt.Println()*/

	crawler.cfg.LogInfo("The crawler has stopped")
}

func (crawler *Crawler) Start(signal chan int) {
	if crawler.Started {
		crawler.cfg.LogInfo("Crawler already started")
		return
	}

	crawler.Signal = signal
	crawler.cfg.LogInfo("Crawler started")
	crawler.Started = true
	crawler.WakeSleepingWorkers()

	defer func() {
		crawler.Started = false

		if e := recover(); e != nil {
			//log and so other stuff
			crawler.cfg.Log("Panic", fmt.Sprintf("%v", e))
		}

		crawler.Quit()

	}()

	for {
		select {
		case worker := <-crawler.WorkerEnded:
			crawler.cfg.LogInfo("Goroutine for host " + worker.String() + " stopped")
			worker.Running = false

			if worker.RecrawlOnFinish {
				worker.RecrawlOnFinish = false
				worker.Recrawl()
			}

			// Pending items aan queue toevoegen, als die er nog zijn
			// zodat we zeker zijn dat de queue leeg is
			worker.EmptyPendingItems()

			// De worker die is gaan slapen terug toevoegen
			// als die nog items heeft, anders stellen we dit uit tot we weer items vinden
			if worker.WantsToGetUp() {
				worker.Sleeping = true
				crawler.SleepingCrawlers.Push(worker)
			}

			// Een worker heeft zich afgesloten
			crawler.distributor.FreeClient(worker.Client)
			worker.Client = nil

			crawler.WakeSleepingWorkers()

		case result := <-crawler.WorkerResult:
			//crawler.cfg.LogInfo("Resultaat van worker ontvangen")
			// Resultaat van een worker verwerken

			// 1. URL's
			for _, url := range result.Links {
				crawler.ProcessUrl(url, result.Source)
			}

			// 2. Andere data (voor later)

			// Kunnen we nieuwe workers wakker maken?
			crawler.WakeSleepingWorkers()

		case worker := <-crawler.WorkerIntroduction:
			if !worker.InRecrawlList {
				crawler.AddRecrawlList(worker)
			}

		case <-crawler.RecrawlTimer:
			crawler.CheckRecrawlList()

			// Zijn er in slaap gebracht die meteen wakker mogen worden gemaakt?
			crawler.WakeSleepingWorkers()

		case code := <-signal:
			if code == 1 {
				return
			}
		}

	}
}

func PrintHeader(header *http.Header) {
	buffer := bytes.NewBufferString("")
	header.Write(buffer)
	fmt.Println(buffer.String())
}
