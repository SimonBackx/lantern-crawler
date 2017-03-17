package config

import (
	"fmt"
)

type CrawlerConfig struct {
	TorProxyAddress *string
}

func (cfg *CrawlerConfig) LogError(err error) {
	fmt.Println("Error:", err)
}
