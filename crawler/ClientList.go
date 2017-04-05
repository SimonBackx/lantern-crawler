package crawler

import (
	"net/http"
)

type ClientItem struct {
	Client *http.Client
	Next   *ClientItem
}

func NewClientItem(client *http.Client) *ClientItem {
	return &ClientItem{Client: client}
}

type ClientList struct {
	First *ClientItem
	Last  *ClientItem
}

func NewClientList() *ClientList {
	return &ClientList{}
}

func (list *ClientList) IsEmpty() bool {
	return list.First == nil
}

func (list *ClientList) Push(client *http.Client) {
	item := NewClientItem(client)

	if list.Last != nil {
		list.Last.Next = item
		list.Last = item
	} else {
		list.Last = item
		list.First = item
	}
}

func (list *ClientList) Pop() *http.Client {
	result := list.First
	if result != nil {
		list.First = result.Next
		if list.First == nil {
			list.Last = nil
		}

		return result.Client
	}
	return nil
}
