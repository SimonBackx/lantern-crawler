package crawler

import (
	"net/http"
	"time"
)

type ClientDistributor interface {
	GetClient() *http.Client
	FreeClient(client *http.Client)
}

type ClearnetDistributor struct {
	Count  int
	Client *http.Client
}

func NewClearnetDistributor() *ClearnetDistributor {
	client := &http.Client{Transport: &http.Transport{}, Timeout: time.Second * 10}
	return &ClearnetDistributor{Client: client, Count: 10}
}

func (dist *ClearnetDistributor) GetClient() *http.Client {
	if dist.Count <= 0 {
		return nil
	}
	dist.Count--
	return dist.Client
}

func (dist *ClearnetDistributor) FreeClient(client *http.Client) {
	dist.Count++
}
