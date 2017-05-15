package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type CrawlerConfig struct {
	UseTorProxy   bool
	OnlyOnion     bool
	LoadFromFiles bool
	MaxDomains    int /// 0 = infinite
	Testing       bool

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
		UseTorProxy:   false,
		OnlyOnion:     false,
		LoadFromFiles: true,
		MaxDomains:    0,

		LogRecrawlingEnabled: false,
		LogGoroutinesEnabled: false,
	}

	file, err := os.Open("/etc/lantern/crawler.conf")
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
