package crawler

import (
	//"net"
	"net/http"
	//"time"
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
	client := &http.Client{
		Timeout: 0,
	}
	return &ClearnetDistributor{Client: client, Count: 50}
}

func (dist *ClearnetDistributor) GetClient() *http.Client {
	if dist.Count <= 0 {
		return nil
	}
	dist.Count--
	return dist.Client /*&http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 200 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   200 * time.Second,
			MaxIdleConnsPerHost:   5000,
			ResponseHeaderTimeout: 200 * time.Second,
		},
		Timeout: time.Second * 200,
	}*/
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
