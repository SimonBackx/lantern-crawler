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
	client := &http.Client{Transport: &http.Transport{}, Timeout: time.Second * 30}
	return &ClearnetDistributor{Client: client, Count: 3}
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

/*if cfg.TorProxyAddress != nil {
	torDialer, err := proxy.SOCKS5("tcp", *cfg.TorProxyAddress, nil, proxy.Direct)

	if err != nil {
		cfg.LogError(err)
		return nil
	}
	transport = &http.Transport{
		Dial: torDialer.Dial,
	}
} else {
	transport = &http.Transport{}
}

client := &http.Client{Transport: transport, Timeout: time.Second * 10}*/
