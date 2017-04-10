package crawler

import (
	"bytes"
	"context"
	"fmt"
	"github.com/SimonBackx/master-project/config"
	//"golang.org/x/net/proxy"
	"net/http"
	"net/url"
	"sync"
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

	WorkerEnded  chan *Hostworker
	WorkerResult chan *WorkerResult

	// Waitgroup die we gebruiken als we op alle requests willen wachten
	waitGroup   sync.WaitGroup
	speedLogger *SpeedLogger
	Stop        chan struct{}

	Started bool
	Signal  chan int
}

func NewCrawler(cfg *config.CrawlerConfig) *Crawler {
	ctx, cancelCtx := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	crawler := &Crawler{cfg: cfg,
		distributor:      NewClearnetDistributor(), //NewTorDistributor(),
		context:          ctx,
		cancelContext:    cancelCtx,
		waitGroup:        wg,
		Workers:          make(map[string]*Hostworker),
		SleepingCrawlers: NewWorkerList(),
		WorkerEnded:      make(chan *Hostworker, 1),
		WorkerResult:     make(chan *WorkerResult, 1),
		speedLogger:      NewSpeedLogger(),
		Stop:             make(chan struct{}, 1),
	}
	crawler.speedLogger.Crawler = crawler

	return crawler
}

func (crawler *Crawler) ProcessUrl(url *url.URL) {
	host := url.Hostname()
	worker := crawler.Workers[host]

	if worker == nil {
		website := &Website{
			URL: host,
		}
		worker = NewHostworker(website, crawler)
		crawler.Workers[host] = worker
	}

	// Crawler queue pushen
	if worker.Running {
		// Pushen d.m.v. channel om concurrency problemen te vermijden
		worker.NewItems.stack(NewCrawlItem(url))
	} else {
		// Geen concurrency problemen mogelijk
		// AddItem kan item ook weggooien als die al gecrawled is
		// Depth = nil, want dit is altijd van een externe host
		worker.AddItem(NewCrawlItem(url), nil)

		if !worker.Sleeping && !worker.Queue.IsEmpty() {
			// Dit domein had geen items, maar nu wel
			worker.Sleeping = true
			crawler.SleepingCrawlers.Push(worker)
		}
	}
}

func (crawler *Crawler) WakeSleepingWorkers() {
	for crawler.SleepingCrawlers.First != nil {
		worker := crawler.SleepingCrawlers.First.Worker

		if worker.Queue.IsEmpty() {
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

func (crawler *Crawler) Panic(str string) {
	crawler.cfg.Log("PANIC", str)
	select {
	case crawler.Signal <- 1:
		break
	default:
		break
	}
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
	}()

	for {
		select {
		case worker := <-crawler.WorkerEnded:
			//crawler.cfg.LogInfo("Goroutine for host " + worker.String() + " stopped")
			worker.Running = false

			// Pending items aan queue toevoegen, als die er nog zijn
			// zodat we zeker zijn dat de queue leeg is
			worker.EmptyPendingItems()

			// De worker die is gaan slapen terug toevoegen
			// als die nog items heeft, anders stellen we dit uit tot we weer items vinden
			if !worker.Queue.IsEmpty() {
				worker.Sleeping = true
				crawler.SleepingCrawlers.Push(worker)
			}

			// Een worker heeft zich afgesloten
			crawler.distributor.FreeClient(worker.Client)
			worker.Client = nil

			crawler.WakeSleepingWorkers()

		case result := <-crawler.WorkerResult:
			// Resultaat van een worker verwerken

			// 1. URL's
			for _, url := range result.Links {
				crawler.ProcessUrl(url)
			}

			// 2. Andere data (voor later)

			// Kunnen we nieuwe workers wakker maken?
			crawler.WakeSleepingWorkers()

		case code := <-signal:
			if code == 1 {
				crawler.cfg.LogInfo("Stopping crawler...")
				crawler.speedLogger.Ticker.Stop()

				close(crawler.Stop)
				crawler.cancelContext()

				// Wacht tot de context is beïndigd
				<-crawler.context.Done()

				// Wachten tot alle goroutines afgelopen zijn die requests verwerken
				crawler.waitGroup.Wait()

				for _, worker := range crawler.Workers {
					fmt.Printf("Remaining queue for %s\n", worker)
					worker.RecrawlQueue.PrintQueue()
				}

				/*for _, domainCrawler := range crawler.DomainCrawlers {
					crawler.cfg.LogInfo(fmt.Sprintf("Queue remaining for %v:", domainCrawler.Website.URL))
					domainCrawler.Queue.PrintQueue()
					fmt.Println()
				}*/

				/*crawler.cfg.LogInfo("Sleeping domains:")
				crawler.SleepingCrawlers.Print()
				fmt.Println()*/

				crawler.cfg.LogInfo("The crawler has stopped")
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
