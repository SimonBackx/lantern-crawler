package crawler

import (
	"bufio"
)

type LeveledQueue struct {
	Levels map[int]*CrawlQueue
}

func NewLeveledQueue() *LeveledQueue {
	return &LeveledQueue{Levels: make(map[int]*CrawlQueue)}
}

func (r *LeveledQueue) Push(item *CrawlItem, level int) {
	if _, present := r.Levels[level]; present == false {
		r.Levels[level] = NewCrawlQueue("Leveled queue")
	}
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

func (l *LeveledQueue) SaveToWriter(writer *bufio.Writer) {
	for _, queue := range l.Levels {
		queue.SaveToWriter(writer)
	}
}
