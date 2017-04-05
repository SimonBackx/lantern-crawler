package crawler

import (
	"fmt"
	"sync"
)

type DomainItem struct {
	DomainCrawler *DomainCrawler
	Next          *DomainItem
	Previous      *DomainItem

	// Pointer vorige naar zichzelf
	Prev_Next **DomainItem

	// Pointer van volgende naar zichzelf
	Next_Prev **DomainItem
}

func (item *DomainItem) Remove() {
	*item.Prev_Next = item.Next
	*item.Next_Prev = item.Previous

	if item.Previous != nil {
		item.Previous.Next_Prev = item.Next_Prev
	}
	if item.Next != nil {
		item.Next.Prev_Next = item.Prev_Next
	}
}

func (item *DomainItem) String() string {
	return item.DomainCrawler.String()
}

func (item *DomainItem) InsertAfter(newItem *DomainItem) {
	newItem.Next = item.Next // Kan nil zijn
	newItem.Previous = item

	*item.Next_Prev = newItem
	item.Next = newItem

	newItem.Next_Prev = item.Next_Prev
	newItem.Prev_Next = &item.Next

	if item.Next != nil {
		item.Next.Prev_Next = &newItem.Next
	}
	item.Next_Prev = &newItem.Previous
}

func NewDomainItem(domainCrawler *DomainCrawler) *DomainItem {
	return &DomainItem{DomainCrawler: domainCrawler}
}

type DomainList struct {
	First *DomainItem
	Last  *DomainItem

	Mutex *sync.Mutex
}

func NewDomainList() *DomainList {
	return &DomainList{Mutex: &sync.Mutex{}}
}

func (list *DomainList) Lock() {
	list.Mutex.Lock()
}

func (list *DomainList) Unlock() {
	list.Mutex.Unlock()
}

func (list *DomainList) Append(domainCrawler *DomainCrawler) {
	list.Lock()
	defer list.Unlock()

	item := NewDomainItem(domainCrawler)

	if list.Last != nil {
		list.Last.InsertAfter(item)
	} else {
		list.Last = item
		list.First = item

		item.Prev_Next = &list.First
		item.Next_Prev = &list.Last
	}
}

func (list *DomainList) Print() {
	item := list.First
	for item != nil {
		fmt.Println(item.String())

		item = item.Next
	}
}

func (list *DomainList) Length() int {
	length := 0
	item := list.First
	for item != nil {
		length++

		item = item.Next
	}
	return length
}
