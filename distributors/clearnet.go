package distributors

import (
	"crypto/tls"
	"net/http"
	"time"
)

type Distributor interface {
	GetClient() *http.Client
	FreeClient(client *http.Client)
	DecreaseClients()
	IncreaseClients()
	AvailableClients() int
	UsedClients() int
}

type Clearnet struct {
	Count    int
	Used     int
	MaxCount int
	Client   *http.Client
}

func NewClearnet(count, max, headerTimeout, requestTimeout int) *Clearnet {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true, // Hmmm?
		//IdleConnTimeout: 15 * time.Second,

		// Tijd dat we wachten op header (zo kort mogelijk houden)
		ResponseHeaderTimeout: time.Duration(headerTimeout) * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Duration(requestTimeout) * time.Second,
		Transport: tr,
	}
	return &Clearnet{Client: client, Count: count, MaxCount: max}
}

func (dist *Clearnet) GetClient() *http.Client {
	if dist.Used >= dist.Count {
		return nil
	}

	dist.Used++
	return dist.Client
}

func (dist *Clearnet) FreeClient(client *http.Client) {
	dist.Used--
}

func (dist *Clearnet) DecreaseClients() {
	if dist.Count < 10 {
		return
	}
	dist.Count = int(float64(dist.Count) * 0.8)
}

func (dist *Clearnet) IncreaseClients() {
	dist.Count = int(float64(dist.Count) * 1.05)
	if dist.Count > dist.MaxCount {
		dist.Count = dist.MaxCount
	}
}

func (dist *Clearnet) AvailableClients() int {
	return dist.Count - dist.Used
}

func (dist *Clearnet) UsedClients() int {
	return dist.Used
}
