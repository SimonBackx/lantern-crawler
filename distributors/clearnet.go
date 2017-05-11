package distributors

import (
	"crypto/tls"
	"net/http"
	"time"
)

type Distributor interface {
	GetClient() *http.Client
	FreeClient(client *http.Client)
}

type Clearnet struct {
	Count  int
	Client *http.Client
}

func NewClearnet() *Clearnet {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true, // Hmmm?
		//IdleConnTimeout: 15 * time.Second,

		// Tijd dat we wachten op header (zo kort mogelijk houden)
		ResponseHeaderTimeout: 15 * time.Second,
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	return &Clearnet{Client: client, Count: 300}
}

func (dist *Clearnet) GetClient() *http.Client {
	if dist.Count <= 0 {
		return nil
	}
	dist.Count--
	return dist.Client
}

func (dist *Clearnet) FreeClient(client *http.Client) {
	dist.Count++
}
