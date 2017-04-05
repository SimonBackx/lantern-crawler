package crawler

import (
	"fmt"
	"time"
)

type SpeedLogger struct {
	Count  int
	Ticker *time.Ticker
}

func NewSpeedLogger() *SpeedLogger {
	logger := &SpeedLogger{Count: 0, Ticker: time.NewTicker(2 * time.Second)}
	go logger.Run()
	return logger
}

func (logger *SpeedLogger) Run() {
	var previousTime *time.Time
	for {
		ti, ok := <-logger.Ticker.C
		if !ok {
			return
		}
		if previousTime == nil {
			previousTime = &ti
			continue
		}

		difference := ti.Sub(*previousTime)
		var requests float64 = float64(logger.Count) / (float64(difference.Nanoseconds()) / 1000000000)

		previousTime = &ti

		fmt.Printf("Req/s = %v\n", requests)
		logger.Count = 0
	}
}

func (logger *SpeedLogger) Log() {
	logger.Count++
}
