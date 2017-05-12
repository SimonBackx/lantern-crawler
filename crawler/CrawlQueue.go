package crawler

import (
	"bufio"
	"fmt"
)

type CrawlQueue struct {
	First  *CrawlItem
	Last   *CrawlItem
	Length int
	Name   string
}

func (q *CrawlQueue) ReadFromReader(reader *bufio.Reader, subdomains []*Subdomain) {
	line, _, _ := reader.ReadLine()
	for len(line) > 0 {
		str := string(line)
		item := NewCrawlItemFromString(&str, subdomains)
		if item != nil {
			q.Push(item)
		} else {
			fmt.Println("Invalid item: " + str)
		}
		line, _, _ = reader.ReadLine()
	}
	// Leest laatste lege lijn ook in
}

func (q *CrawlQueue) SaveToWriter(writer *bufio.Writer) {
	item := q.First
	for item != nil {
		writer.WriteString(item.SaveToString())
		writer.WriteString("\n")
		item = item.Next
	}
	writer.WriteString("\n")
}

func (q *CrawlQueue) IsEqual(b *CrawlQueue) bool {
	if q.Length != b.Length {
		return false
	}
	if q.IsEmpty() && b.IsEmpty() {
		return true
	}

	item1 := q.First
	item2 := b.First

	for item1 != nil && item2 != nil {
		if !item1.IsEqual(item2) {
			return false
		}

		item1 = item1.Next
		item2 = item2.Next
	}

	return true
}

func NewCrawlQueue(name string) *CrawlQueue {
	return &CrawlQueue{Name: name}
}

func (queue *CrawlQueue) Clear() {
	item := queue.First
	for item != nil {
		item.Previous = nil

		backup := item.Next
		item.Next = nil
		item.Queue = nil

		item = backup
	}

	queue.First = nil
	queue.Last = nil
	queue.Length = 0
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

	queue.Length--
	return item
}

func (queue *CrawlQueue) Push(item *CrawlItem) {
	if item.Queue != nil {
		fmt.Println("PANIC! pushing on queue " + queue.Name + ", already on queue " + item.Queue.Name)
		return
	}
	item.Queue = queue
	item.Next = nil
	item.Previous = nil
	queue.Length++

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
		fmt.Println("Panic: Wil item verwijderen uit foute queue (" + queue.Name + "), maar is in " + item.Queue.Name)
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
	queue.Length--
}

func (queue *CrawlQueue) String() string {
	if queue == nil {
		return "<nil>"
	}
	return queue.Name
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
		queue.Length = q.Length
		return
	}

	queue.Length += q.Length
	queue.Last.Next = q.First

	if q.First != nil {
		q.First.Previous = queue.Last
	}

	queue.Last = q.Last
}
