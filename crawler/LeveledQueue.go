package crawler

import (
	"bufio"
)

type LeveledQueue struct {
	Levels []*CrawlQueue
}

func NewLeveledQueue() *LeveledQueue {
	lvls := make([]*CrawlQueue, maxFailCount+1)
	for i := 0; i <= maxFailCount; i++ {
		lvls[i] = NewCrawlQueue("Leveled queue")
	}
	return &LeveledQueue{Levels: lvls}
}

func (r *LeveledQueue) Push(item *CrawlItem, level int) {
	r.Levels[level].Push(item)
}

// Popt de eerste uit de leveled queue
func (r *LeveledQueue) Pop() *CrawlItem {
	for _, queue := range r.Levels {
		// Onderstanade check heeft nog wat meer abstractie nodig
		if queue.First != nil && queue.First.NeedsRetry() {
			return queue.Pop()
		}
	}

	return nil
}

func (r *LeveledQueue) First() *CrawlItem {
	for _, queue := range r.Levels {
		if queue.First != nil {
			return queue.First
		}
	}

	return nil
}

func (l *LeveledQueue) IsEqual(b *LeveledQueue) bool {
	for i := 0; i <= maxFailCount; i++ {
		if !l.Levels[i].IsEqual(b.Levels[i]) {
			return false
		}
	}
	return true
}

func (l *LeveledQueue) ReadFromReader(reader *bufio.Reader, subdomains []*Subdomain) {
	for i := 0; i <= maxFailCount; i++ {
		queue := NewCrawlQueue("Leveled queue")
		queue.ReadFromReader(reader, subdomains)
		l.Levels[i] = queue
	}
}

func (l *LeveledQueue) SaveToWriter(writer *bufio.Writer) {
	for _, queue := range l.Levels {
		queue.SaveToWriter(writer)
	}
}
