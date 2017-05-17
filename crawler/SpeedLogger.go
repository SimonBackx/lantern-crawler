package crawler

import (
	"fmt"
	"github.com/SimonBackx/lantern-crawler/queries"
	"runtime"
	"time"
)

type SpeedLogger struct {
	Count        int
	DownloadSize int64
	DownloadTime time.Duration
	Timeouts     int

	Ticker  *time.Ticker
	Crawler *Crawler
}

func NewSpeedLogger() *SpeedLogger {
	logger := &SpeedLogger{Count: 0, Ticker: time.NewTicker(60 * time.Second)}
	go logger.Run()
	return logger
}

func (logger *SpeedLogger) Run() {
	//var previousTime *time.Time
	var m runtime.MemStats

	for {
		_, ok := <-logger.Ticker.C
		if !ok {
			return
		}

		var requests float64 = float64(logger.Count)
		workers := logger.Crawler.distributor.UsedClients()
		domains := len(logger.Crawler.Workers)

		downloadSpeed := int(float64(logger.DownloadSize) / 60 / 1024) // * 6
		downloadSize := 0
		downloadTime := 0

		if logger.Count > 0 {
			downloadSize = int(float64(logger.DownloadSize) / 1024 / float64(logger.Count))
			downloadTime = int(logger.DownloadTime.Seconds() * 1000 / float64(logger.Count))
		}

		runtime.ReadMemStats(&m)
		memoryAlloc := m.Alloc / 1024
		memorySys := m.Sys / 1024

		logger.Crawler.cfg.Log("Stat", fmt.Sprintf("%v requests, %v workers, %v domains, %v sleeping, %v KB/s, %v KB/page, %v ms/page, %v timeouts, %v KB alloc, %v KB sys",
			requests,
			workers,
			domains,
			logger.Crawler.SleepingCrawlers.Length(),
			downloadSpeed,
			downloadSize,
			downloadTime,
			logger.Timeouts,
			memoryAlloc,
			memorySys,
		))

		next := logger.Crawler.GetNextRecrawlDuration()
		if next == nil {
			logger.Crawler.cfg.LogInfo("Next recrawl unknown")
		} else {
			logger.Crawler.cfg.LogInfo(fmt.Sprintf("Next recrawl in %v minutes", next.Minutes()))
		}

		// Als er veel timeouts zijn -> vertragen
		if logger.Timeouts > logger.Crawler.cfg.MaxTimeouts && logger.Crawler.distributor.AvailableClients() >= 0 {
			logger.Crawler.distributor.DecreaseClients()
		} else if logger.Timeouts < logger.Crawler.cfg.MinTimeouts && logger.Crawler.distributor.AvailableClients() == 0 {
			logger.Crawler.distributor.IncreaseClients()
		}

		stats := queries.NewStats(logger.Count, logger.Timeouts, workers, domains, downloadSpeed, downloadTime, downloadSize, memoryAlloc, memorySys)
		logger.Crawler.ApiController.SaveStats(stats)

		logger.Count = 0
		logger.DownloadSize = 0
		logger.DownloadTime = 0

		logger.Timeouts = 0

	}
}

func (logger *SpeedLogger) Log(duration time.Duration, bytes int) {
	logger.Count++
	logger.DownloadSize += int64(bytes)
	logger.DownloadTime += duration
}

func (logger *SpeedLogger) LogTimeout() {
	logger.Timeouts++
}
