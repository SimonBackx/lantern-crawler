package crawler

import (
	"fmt"
)

type WorkerItem struct {
	Worker *Hostworker
	Next   *WorkerItem
}

func (item *WorkerItem) String() string {
	return item.Worker.String()
}

func NewWorkerItem(worker *Hostworker) *WorkerItem {
	return &WorkerItem{Worker: worker}
}

type WorkerList struct {
	First *WorkerItem
	Last  *WorkerItem
}

func NewWorkerList() *WorkerList {
	return &WorkerList{}
}

func (list *WorkerList) IsEmpty() bool {
	return list.First == nil
}

func (list *WorkerList) Push(worker *Hostworker) {
	item := NewWorkerItem(worker)

	if list.Last != nil {
		list.Last.Next = item
		list.Last = item
	} else {
		list.Last = item
		list.First = item
	}
}

/*func (list *WorkerList) PushSorted(worker *Hostworker) {
	if worker.IntroductionPoints.IsEmpty() {
		return
	}

	item := NewWorkerItem(worker)

	if list.First != nil {
		// Overlopen en sorteren op lastDownload van het eerste inroduction point.
		// Volgorde van oud > nieuw
		var prev *WorkerItem
		next := list.First
		for next != nil && next.Worker.IntroductionPoints.First.LastDownload.Before(worker.IntroductionPoints.First.LastDownload) {
			prev = next
			next = next.Next
		}

		if prev == nil {
			list.First = item
		} else {
			prev.Next = item
		}
		item.Next = next

		if next == nil {
			list.Last = item
		}
	} else {
		// Gewoon toevoegen als enige item
		list.Last = item
		list.First = item
	}
}*/

func (list *WorkerList) Pop() *Hostworker {
	result := list.First
	if result != nil {
		list.First = result.Next
		if list.First == nil {
			list.Last = nil
		}

		return result.Worker
	}
	return nil
}

func (list *WorkerList) Print() {
	item := list.First
	for item != nil {
		fmt.Println(item.String())

		item = item.Next
	}
}

func (list *WorkerList) Length() int {
	length := 0
	item := list.First
	for item != nil {
		length++

		item = item.Next
	}
	return length
}
