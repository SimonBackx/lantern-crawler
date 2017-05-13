package crawler

import (
	"fmt"
	"github.com/SimonBackx/lantern-crawler/queries"
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
	for {
		_, ok := <-logger.Ticker.C
		if !ok {
			return
		}

		var requests float64 = float64(logger.Count)
		workers := logger.Crawler.distributor.UsedClients()

		domains := len(logger.Crawler.Workers)
		logger.Crawler.cfg.Log("Stat", fmt.Sprintf("%v requests, %v workers, %v domains", int(requests/60), workers, domains))

		downloadSpeed := int(float64(logger.DownloadSize) / 60000) // * 6
		downloadSize := 0
		downloadTime := 0

		if logger.Count > 0 {
			downloadSize = int(float64(logger.DownloadSize) / 1000 / float64(logger.Count))
			downloadTime = int(logger.DownloadTime.Seconds() * 1000 / float64(logger.Count))

			logger.Crawler.cfg.Log("Stat", fmt.Sprintf("%v KB/s, %v KB/page, %v ms/page, %v timeouts",
				downloadSpeed,
				downloadSize,
				downloadTime,
				logger.Timeouts,
			))
		}

		// Als er veel timeouts zijn -> vertragen
		if logger.Timeouts > 60 && logger.Crawler.distributor.AvailableClients() >= 0 {
			logger.Crawler.distributor.DecreaseClients()
		} else if logger.Timeouts < 20 && logger.Crawler.distributor.AvailableClients() == 0 {
			logger.Crawler.distributor.IncreaseClients()
		}

		stats := queries.NewStats(logger.Count, logger.Timeouts, workers, domains, downloadSpeed, downloadTime, downloadSize)
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
