package crawler

import (
	"context"
	"github.com/SimonBackx/lantern-crawler/distributors"
	"github.com/SimonBackx/lantern-crawler/queries"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Crawler struct {
	cfg           *CrawlerConfig
	distributor   distributors.Distributor
	context       context.Context
	cancelContext context.CancelFunc
	ApiController *ApiController

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

	// General update timer
	UpdateTimer <-chan time.Time

	WorkerEnded        WorkerChannel
	WorkerResult       WorkerResultChannel
	WorkerIntroduction WorkerChannel

	// Waitgroup die we gebruiken als we op alle requests willen wachten
	waitGroup   sync.WaitGroup
	speedLogger *SpeedLogger
	Stop        chan struct{}

	Started bool
	Signal  chan int
	Queries []queries.Query
}

func NewCrawler(cfg *CrawlerConfig) *Crawler {
	ctx, cancelCtx := context.WithCancel(context.Background())

	var distributor distributors.Distributor
	if cfg.UseTorProxy {
		distributor = distributors.NewTor(cfg.TorDaemons, cfg.InitialWorkers, cfg.MaxWorkers)
	} else {
		distributor = distributors.NewClearnet(cfg.InitialWorkers, cfg.MaxWorkers)
	}

	var wg sync.WaitGroup
	crawler := &Crawler{cfg: cfg,
		distributor:        distributor,
		context:            ctx,
		cancelContext:      cancelCtx,
		waitGroup:          wg,
		Workers:            make(map[string]*Hostworker),
		SleepingCrawlers:   NewWorkerList(),
		RecrawlList:        NewWorkerList(),
		WorkerEnded:        NewWorkerChannel(),
		WorkerResult:       NewWorkerResultChannel(),
		WorkerIntroduction: NewWorkerChannel(),
		speedLogger:        NewSpeedLogger(),
		Stop:               make(chan struct{}, 1),
		RecrawlTimer:       make(<-chan time.Time, 1),
		UpdateTimer:        make(<-chan time.Time, 1),
		Queries:            make([]queries.Query, 0),
		ApiController:      NewApiController(),
	}
	crawler.speedLogger.Crawler = crawler
	if !cfg.Testing {
		crawler.RefreshQueries()
	}

	// Nieuwe queries etc laden
	crawler.UpdateTimer = time.After(time.Minute * 5)

	if !cfg.LoadFromFiles {
		return crawler
	}

	cfg.LogInfo("Loading hosts from disk...")

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
			splitted := strings.Split(worker.Host, ".")

			if cfg.OnlyOnion {
				if len(splitted) != 1 {
					continue
				}
			} else {
				if len(splitted) != 2 {
					continue
				}
			}
			crawler.Workers[worker.Host] = worker

			if worker.WantsToGetUp() {
				worker.Sleeping = true
				crawler.SleepingCrawlers.Push(worker)
			}
		}
	}
	cfg.LogInfo("Done.")

	return crawler
}

func (crawler *Crawler) RefreshQueries() {
	queries, err := crawler.ApiController.GetQueries()
	if err != nil {
		crawler.cfg.LogError(err)
		return
	}
	crawler.Queries = queries
}

func (crawler *Crawler) GetDomainForUrl(splitted []string) string {
	if crawler.cfg.OnlyOnion {
		return splitted[len(splitted)-2]
	} else {
		return splitted[len(splitted)-2] + "." + splitted[len(splitted)-1]
	}
}

func (crawler *Crawler) ProcessUrl(u *url.URL) {
	host := crawler.GetDomainForUrl(strings.Split(u.Host, "."))
	worker := crawler.Workers[host]

	if worker == nil {
		if crawler.cfg.MaxDomains != 0 && len(crawler.Workers) >= crawler.cfg.MaxDomains {
			return
		}

		worker = NewHostworker(host, crawler)
		crawler.Workers[host] = worker
	}

	// Crawler queue pushen
	if worker.Running {
		// Pushen d.m.v. channel om concurrency problemen te vermijden
		worker.NewItems.stack(u)

	} else {
		// Geen concurrency problemen mogelijk
		// NewReference kan url ook weggooien als die al gecrawled is
		// Depth = nil, want dit is altijd van een externe host
		worker.NewReference(u, nil, false)

		if !worker.Sleeping && worker.WantsToGetUp() {
			// Dit domein had geen items, maar nu wel
			worker.Sleeping = true
			crawler.SleepingCrawlers.Push(worker)
		}
	}
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
	if duration < 0 {
		duration = 0
	}

	// Minimaal 5 seconden wachten en zoveel mogelijk combineren door
	// 5 seconden langer te wachten
	crawler.RecrawlTimer = time.After(duration + time.Second*5)
}

func (crawler *Crawler) AddRecrawlList(worker *Hostworker) {
	if crawler.cfg.LogRecrawlingEnabled {
		crawler.cfg.LogInfo("Added to recrawl list: " + worker.String())
	}

	if crawler.RecrawlList.IsEmpty() {
		crawler.SetRecrawlFirst(worker)
	}

	crawler.RecrawlList.Push(worker)
	worker.InRecrawlList = true
}

func (crawler *Crawler) CheckRecrawlList() {
	if crawler.cfg.LogRecrawlingEnabled {
		crawler.cfg.LogInfo("Check recrawl list")
	}

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

	crawler.cfg.LogInfo("Stopping context...")
	crawler.cancelContext()

	// Wacht tot de context is beïndigd
	<-crawler.context.Done()

	// Wachten tot alle goroutines afgelopen zijn die requests verwerken
	crawler.cfg.LogInfo("Stopping goroutines...")
	crawler.waitGroup.Wait()

	crawler.cfg.LogInfo("Saving progress...")
	for _, worker := range crawler.Workers {
		if worker.NeedsWriteToDisk() {
			worker.MoveToDisk()
		}
	}

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
			crawler.cfg.Log("Panic", identifyPanic())
		}

		crawler.Quit()

	}()

	for {
		select {
		case workers := <-crawler.WorkerEnded:
			for _, worker := range workers {
				if crawler.cfg.LogGoroutinesEnabled {
					crawler.cfg.LogInfo("Goroutine for host " + worker.String() + " stopped")
				}

				worker.Running = false

				if worker.RecrawlOnFinish {
					worker.RecrawlOnFinish = false
					worker.Recrawl()
				}

				// Pending items aan queue toevoegen, als die er nog zijn
				// zodat we zeker zijn dat de queue leeg is

				// De worker die is gaan slapen terug toevoegen
				// als die nog items heeft, anders stellen we dit uit tot we weer items vinden
				if worker.WantsToGetUp() {
					worker.Sleeping = true
					crawler.SleepingCrawlers.Push(worker)
				} else {
					// todo: toevoegen aan completeFails?
				}

				// Een worker heeft zich afgesloten
				crawler.distributor.FreeClient(worker.Client)
				worker.Client = nil
			}

			crawler.WakeSleepingWorkers()

		case result := <-crawler.WorkerResult:
			// Resultaat van een of meerdere workers verwerken

			// 1. URL's
			for _, url := range result.Links {
				crawler.ProcessUrl(url)
			}

			// 2. Andere data (voor later)

			// Kunnen we nieuwe workers wakker maken?
			crawler.WakeSleepingWorkers()

		case workers := <-crawler.WorkerIntroduction:
			for _, worker := range workers {
				if !worker.InRecrawlList {
					crawler.AddRecrawlList(worker)
				}
			}

		case <-crawler.UpdateTimer:
			crawler.RefreshQueries()
			crawler.UpdateTimer = time.After(time.Minute * 5)

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
