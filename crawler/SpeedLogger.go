package crawler

import (
	"fmt"
	"time"
)

type SpeedLogger struct {
	Count   int
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

		//previousTime = &ti

		logger.Crawler.cfg.Log("STATS", fmt.Sprintf("%v REQ/S (%v unique domains known)", requests, len(logger.Crawler.Workers)))
		logger.Count = 0
	}
}

func (logger *SpeedLogger) Log() {
	logger.Count++
}
