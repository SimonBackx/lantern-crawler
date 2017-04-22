package crawler

import (
	"fmt"
	"time"
)

type SpeedLogger struct {
	Count                int
	UnavailableCount     int
	SuccessfulRetryCount int

	/// Amount of URL's that successfully recrawled
	RecrawlCount int

	/// Amount of new discovered URL's
	NewURLsCount int

	/// Amount of items added to the priority queue
	NewPriorityQueue    int
	NewQueue            int
	NewLowPriorityQueue int
	NewFailedQueue      int
	Timeouts            int
	/// Amount of items that switch to priority queue and were priviously on a lower
	/// priority queue
	SwitchesToPriority int

	PoppedFromPriorityQueue    int
	PoppedFromQueue            int
	PoppedFromLowPriorityQueue int
	PoppedFromFailedQueue      int

	Ticker  *time.Ticker
	Crawler *Crawler
}

func NewSpeedLogger() *SpeedLogger {
	logger := &SpeedLogger{Count: 0, Ticker: time.NewTicker(10 * time.Second)}
	go logger.Run()
	return logger
}

func (logger *SpeedLogger) Run() {
	//var previousTime *time.Time
	for {
		_, ok := <-logger.Ticker.C
		if !ok {
			return
		}
		/*if previousTime == nil {
			previousTime = &ti
			continue
		}*/

		//difference := ti.Sub(*previousTime)
		var requests float64 = float64(logger.Count) / 10
		var unavailable float64 = float64(logger.UnavailableCount) / 10
		var sucretry float64 = float64(logger.SuccessfulRetryCount) / 10

		//previousTime = &ti
		domains := len(logger.Crawler.Workers)
		sleeping := logger.Crawler.SleepingCrawlers.Length()
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("%v REQ/S, %v UNAVAILABLE/S, %v SUCCESSFUL RETRIES/S, %v DOMAINS, %v SLEEPING", requests, unavailable, sucretry, domains, sleeping))
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("%v NEW URL's, %v RECRAWLS, %v TIMEOUTS", logger.NewURLsCount, logger.RecrawlCount, logger.Timeouts))
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("Priority Queue		+%v	-%v excl. %v switches", logger.NewPriorityQueue, logger.PoppedFromPriorityQueue, logger.SwitchesToPriority))
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("Queue			+%v	-%v", logger.NewQueue, logger.PoppedFromQueue))
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("Low-priority Queue	+%v	-%v", logger.NewLowPriorityQueue, logger.PoppedFromLowPriorityQueue))
		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("Failed Queue 		+%v	-%v", logger.NewFailedQueue, logger.PoppedFromFailedQueue))

		logger.Count = 0
		logger.Timeouts = 0
		logger.UnavailableCount = 0
		logger.SuccessfulRetryCount = 0

		logger.RecrawlCount = 0
		logger.NewURLsCount = 0
		logger.NewPriorityQueue = 0
		logger.NewQueue = 0
		logger.NewLowPriorityQueue = 0
		logger.NewFailedQueue = 0

		logger.SwitchesToPriority = 0
		logger.PoppedFromPriorityQueue = 0
		logger.PoppedFromQueue = 0
		logger.PoppedFromLowPriorityQueue = 0
		logger.PoppedFromFailedQueue = 0
	}
}

func (logger *SpeedLogger) Log() {
	logger.Count++
}

func (logger *SpeedLogger) LogSuccessfulRetry() {
	logger.SuccessfulRetryCount++
}

func (logger *SpeedLogger) LogUnavailable() {
	logger.UnavailableCount++
}
