package config

import (
	"fmt"
	"time"
)

type CrawlerConfig struct {
	UseTorProxy   bool
	OnlyOnion     bool
	LoadFromFiles bool
	SaveToFiles   bool
	MaxDomains    int /// 0 = infinite

	LogRecrawlingEnabled bool
	LogGoroutinesEnabled bool
}

func (cfg *CrawlerConfig) LogError(err error) {
	cfg.Log("Error", err.Error())
}

func (cfg *CrawlerConfig) LogInfo(str string) {
	cfg.Log("Info", str)
}

func (cfg *CrawlerConfig) Log(label, str string) {
	t := time.Now()
	fmt.Printf("[%v: %v] %v\n", label, t.Format("15:04:05.000"), str)
}
