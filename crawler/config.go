package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type CrawlerConfig struct {
	UseTorProxy      bool
	OnlyOnion        bool
	LoadFromFiles    bool
	MaxDomains       int /// 0 = infinite
	Testing          bool
	MaxTimeouts      int
	MinTimeouts      int
	MaxWorkers       int
	InitialWorkers   int
	TorDaemons       int
	SleepAfter       int
	SleepAfterRandom int

	SleepTime       int
	SleepTimeRandom int

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

func ConfigFromFile() *CrawlerConfig {
	// Default configuration
	cfg := &CrawlerConfig{
		UseTorProxy:    false,
		OnlyOnion:      false,
		LoadFromFiles:  true,
		MaxDomains:     0,
		MinTimeouts:    15,
		MaxTimeouts:    50,
		MaxWorkers:     1000,
		InitialWorkers: 560,
		TorDaemons:     20,

		SleepAfter:       10,
		SleepAfterRandom: 50,
		SleepTime:        4000,
		SleepTimeRandom:  4000,

		LogRecrawlingEnabled: false,
		LogGoroutinesEnabled: false,
	}

	defer func() {
		file, err := os.Create("/etc/lantern/crawler.json")
		if err == nil {
			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "    ")
			encoder.Encode(cfg)
		}
	}()

	file, err := os.Open("/etc/lantern/crawler.json")
	if err != nil {
		cfg.LogInfo("Using default configuration")
		return cfg
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	err = decoder.Decode(cfg)
	if err != nil {
		cfg.LogError(err)
	}
	return cfg
}

func (cfg *CrawlerConfig) Describe() {
	if !cfg.LoadFromFiles {
		cfg.LogInfo("LoadFromFiles disabled")
	}

	if cfg.MaxDomains != 0 {
		cfg.LogInfo(fmt.Sprintf("MaxDomains = %v", cfg.MaxDomains))
	}

	if cfg.UseTorProxy {
		cfg.LogInfo("Crawling tor")
		if !cfg.OnlyOnion {
			cfg.Log("Warning", "OnlyOnion disabled")
		}
	} else {
		cfg.LogInfo("Crawling clearweb")
		if cfg.OnlyOnion {
			cfg.Log("Warning", "OnlyOnion enabled")
		}
	}

	if cfg.LogRecrawlingEnabled {
		cfg.LogInfo("LogRecrawlingEnabled")
	}

	if cfg.LogGoroutinesEnabled {
		cfg.LogInfo("LogGoroutinesEnabled")
	}
}
