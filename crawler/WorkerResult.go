package crawler

import (
	"net/url"
)

type WorkerResult struct {
	Links  []*url.URL
	Source *url.URL
}

func NewWorkerResult() *WorkerResult {
	return &WorkerResult{
		Links: make([]*url.URL, 0, 5),
	}
}

func (r *WorkerResult) Append(url *url.URL) {
	r.Links = append(r.Links, url)
}
