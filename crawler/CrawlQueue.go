package crawler

import "fmt"

type CrawlQueue struct {
	First *CrawlItem
	Last  *CrawlItem
}

func NewCrawlQueue() *CrawlQueue {
	return &CrawlQueue{}
}

func (queue *CrawlQueue) Clear() {
	queue.First = nil
	queue.Last = nil
}

func (queue *CrawlQueue) Pop() *CrawlItem {
	if queue == nil || queue.First == nil {
		return nil
	}

	item := queue.First
	queue.First = queue.First.Next

	if item.Next != nil {
		// Voor het wijzigen van next moeten we ook altijd previous wijzigen!
		item.Next.Previous = nil
	} else {
		// We hebben de laatste gepopt
		// dan verwijderen we ook de verwijzing naar de laatste
		queue.Last = nil
	}

	item.Next = nil
	// Previous op nil zetten is niet nodig, dat is al zo als het item eerst in de queue staat
	item.Queue = nil

	return item
}

func (queue *CrawlQueue) Push(item *CrawlItem) {
	item.Next = nil
	item.Previous = nil

	if queue.First == nil {
		// Eerst in de queue zetten
		queue.First = item
		queue.Last = item
		return
	}

	queue.Last.Next = item

	// Previous van nieuwe item laten wijzen naar het huidige laatste item
	item.Previous = queue.Last

	// Nieuwe item instellen als laatste item
	queue.Last = item
}

/**
 * Verwijder een item uit deze queue. Gebruik bij voorkeur Pop(), die is iets performanter
 */
func (queue *CrawlQueue) Remove(item *CrawlItem) {
	if item.Queue != queue {
		// Uitschrijven voor de zekerheid, panic gaat soms fout bij goroutines
		fmt.Println("Panic: Wil item verwijderen uit foute queue")
		panic("Wil item verwijderen uit foute queue")
	}

	if item.Previous != nil {
		item.Previous.Next = item.Next
	} else {
		// Dit item heeft geen vorige, en is dus eerst
		if queue.First == item {
			// Check is eigenlijk onnodig, tensij bij fout gebruik van de methode
			queue.First = item.Next
		}
	}

	if item.Next != nil {
		item.Next.Previous = item.Previous
	} else {
		// Dit item heeft geen volgende, en is dus laatst
		if queue.Last == item {
			// Check is eigenlijk onnodig, tensij bij fout gebruik van de methode
			queue.Last = item.Previous
		}
	}

	item.Next = nil
	item.Previous = nil
	item.Queue = nil
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

func (queue *CrawlQueue) Top() *CrawlItem {
	return queue.First
}

func (queue *CrawlQueue) Merge(q *CrawlQueue) {
	if queue.First == nil {
		queue.First = q.First
		queue.Last = q.Last
		return
	}

	queue.Last.Next = q.First

	if q.First != nil {
		q.First.Previous = queue.Last
	}

	queue.Last = q.Last
}
