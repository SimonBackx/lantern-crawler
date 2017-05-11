package queries

import (
	"time"
)

type Stats struct {
	Date          time.Time `json:"date" bson:"date"`
	Requests      int       `json:"requests" bson:"requests"`
	Timeouts      int       `json:"timeouts" bson:"timeouts"`
	Workers       int       `json:"workers" bson:"workers"`
	Domains       int       `json:"domains" bson:"domains"`
	DownloadSpeed int       `json:"downloadSpeed" bson:"downloadSpeed"`
	DownloadTime  int       `json:"downloadTime" bson:"downloadTime"`
	DownloadSize  int       `json:"downloadSize" bson:"downloadSize"`
}

func NewStats(requests, timeouts, workers, domains, downloadSpeed, downloadTime, downloadSize int) *Stats {
	return &Stats{Date: time.Now(), Requests: requests, Timeouts: timeouts, Workers: workers, Domains: domains, DownloadSpeed: downloadSpeed, DownloadTime: downloadTime, DownloadSize: downloadSize}
}

func AverageStats(stats []*Stats) *Stats {
	result := NewStats(0, 0, 0, 0, 0, 0, 0)
	for _, stat := range stats {
		result.Requests += stat.Requests
		result.Timeouts += stat.Timeouts
		result.Workers += stat.Workers
		result.Domains += stat.Domains
		result.DownloadSpeed += stat.DownloadSpeed
		result.DownloadTime += stat.DownloadTime
		result.DownloadSize += stat.DownloadSize
	}

	if len(stats) == 0 {
		return result
	}

	result.Requests /= len(stats)
	result.Timeouts /= len(stats)
	result.Workers /= len(stats)
	result.Domains /= len(stats)
	result.DownloadSpeed /= len(stats)
	result.DownloadTime /= len(stats)
	result.DownloadSize /= len(stats)

	result.Date = stats[len(stats)/2].Date

	return result
}
