package crawler

import "fmt"

type CrawlQueue struct {
	First *CrawlItem
	Last  *CrawlItem
}

func NewCrawlQueue() *CrawlQueue {
	return &CrawlQueue{}
}

func (queue *CrawlQueue) Pop() *CrawlItem {
	if queue == nil || queue.First == nil {
		return nil
	}

	item := queue.First
	queue.First = queue.First.Next
	item.Next = nil

	// Als we de laatste gepopt hebben,
	// dan verwijderen we ook de verwijzing naar de laatste
	if queue.First == nil {
		queue.Last = nil
	}

	return item
}

func (queue *CrawlQueue) Push(item *CrawlItem) {
	item.Next = nil

	if queue.First == nil {
		queue.First = item
		queue.Last = item
		return
	}

	queue.Last.Next = item
	queue.Last = item
}

func (queue *CrawlQueue) IsEmpty() bool {
	return queue == nil || queue.First == nil
}

func (queue *CrawlQueue) PrintQueue() {
	item := queue.First
	for item != nil {
		fmt.Println(item.String())

		item = item.Next
	}
}
