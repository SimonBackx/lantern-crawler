package crawler

import (
	"fmt"
)

const maxRecrawlDepth = 5

/**
 * Een recrawl queue houdt referenties bij naar de pagina's van een website
 * met een maximale diepte. Deze moeten regelmatig opnieuw gecrawld worden,
 * om zo nieuwe URL's snel te kunnen ontdekken.
 */
type RecrawlQueue struct {
	// Voor elke diepte is er een verschillende crawlqueue, deze bevat telkens
	// een lijst met pagina's die reeds gedownload werden. Deze zijn gerangschikt zodat
	// de pagina's die het langs geleden gedownload werden vooraan staan
	// Deze rangschikking is niet gegarandeerd en zal voor performantie soms genegeerd worden
	// bij het aanpassen van depths. Dit kan enkel gebeuren bij toevoeging van een item dat eigenlijk
	// vroeger in de rij zou moeten staan, en hierdoor bereid is om pas later opnieuw gecrawld te worden
	// dan het eigenlijk zou horen
	Depths map[int]*CrawlQueue
}

func NewRecrawlQueue() *RecrawlQueue {
	return &RecrawlQueue{Depths: make(map[int]*CrawlQueue)}
}

func (r *RecrawlQueue) Push(item *CrawlItem) {
	if item.Depth > maxRecrawlDepth {
		return
	}
	if _, present := r.Depths[item.Depth]; present == false {
		r.Depths[item.Depth] = NewCrawlQueue()
	}
	r.Depths[item.Depth].Push(item)
}

func (r *RecrawlQueue) Pop() *CrawlItem {
	for _, queue := range r.Depths {
		if queue.First != nil && queue.First.NeedsRecrawl() {
			return queue.Pop()
		}
	}

	return nil
}

func (r *RecrawlQueue) DepthUpdated(item *CrawlItem) {
	if item.LastDownload == nil {
		// Nog nooit gedownload, staat niet in de recrawl queue
		return
	}
	if item.Depth > maxRecrawlDepth {
		return
	}
	// Verwijderen uit huidige queue
	item.Remove()

	// Toevoegen in nieuwe
	r.Push(item)
}

func (r *RecrawlQueue) GetQueue(depth int) *CrawlQueue {
	return r.Depths[depth]
}

func (r *RecrawlQueue) PrintQueue() {
	for depth, queue := range r.Depths {
		fmt.Printf("Diepte %v:\n", depth)
		queue.PrintQueue()
	}
}
