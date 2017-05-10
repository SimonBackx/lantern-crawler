package crawler

import (
	"net/url"
	"testing"
)

func TestCrawlQueue(test *testing.T) {
	u, _ := url.Parse("/hello")
	queue := NewCrawlQueue("test")
	item1 := NewCrawlItem(u)
	item2 := NewCrawlItem(u)
	item3 := NewCrawlItem(u)
	item4 := NewCrawlItem(u)

	queue.Push(item1)

	if queue.First != item1 || queue.Last != item1 {
		test.Log("Queue first / last not set correctly")
		test.Fail()
	}

	if item1.Previous != nil || item1.Next != nil || item1.Queue != queue {
		test.Log("Item previous, next or queue not set correctly")
		test.Fail()
	}

	queue.Push(item2)

	if queue.First != item1 || queue.Last != item2 {
		test.Log("Queue first / last not set correctly (2)")
		test.Fail()
	}

	if item1.Next != item2 || item2.Previous != item1 || item1.Previous != nil || item2.Next != nil || item1.Queue != queue || item2.Queue != queue {
		test.Log("Item previous, next or queue not set correctly (2)")
		test.Fail()
	}

	queue.Push(item3)

	if queue.First != item1 || queue.Last != item3 {
		test.Log("Queue first / last not set correctly (3)")
		test.Fail()
	}

	if item1.Next != item2 || item2.Previous != item1 || item1.Previous != nil || item2.Next != item3 || item3.Previous != item2 || item3.Next != nil || item1.Queue != queue || item2.Queue != queue || item3.Queue != queue {
		test.Log("Item previous, next or queue not set correctly (3)")
		test.Fail()
	}

	popped := queue.Pop()

	if popped != item1 {
		test.Log("Pop failed")
		test.Fail()
	}

	if queue.First != item2 || queue.Last != item3 {
		test.Log("Queue first / last not set correctly (4)")
		test.Fail()
	}

	if item1.Next != nil || item1.Previous != nil || item2.Queue != queue || item3.Queue != queue {
		test.Log("Item previous, next or queue not set correctly (4)")
		test.Fail()
	}

	if item2.Previous != nil || item2.Next != item3 || item3.Previous != item2 || item3.Next != nil {
		test.Log("Item previous, next not set correctly (5)")
		test.Fail()
	}

	queue.Push(item4)

	item3.Remove()

	if queue.First != item2 || queue.Last != item4 {
		test.Log("Queue first / last not set correctly (5)")
		test.Fail()
	}

	if item3.Queue != nil || item2.Queue != queue || item4.Queue != queue || item2.Previous != nil || item2.Next != item4 || item4.Previous != item2 || item3.Next != nil || item3.Previous != nil {
		test.Log("Item previous, next or queue not set correctly (6)")
		test.Fail()
	}

	queue.Clear()

	if queue.First != nil || queue.Last != nil || item2.Queue != nil || item3.Queue != nil {
		test.Log("Queue first / last not set correctly (6)")
		test.Fail()
	}

	queue.Push(item2)

	if queue.First != item2 || queue.Last != item2 {
		test.Log("Queue first / last not set correctly (7)")
		test.Fail()
	}

	if item2.Previous != nil || item2.Next != nil || item2.Queue != queue {
		test.Log("Item previous, next or queue not set correctly (7)")
		test.Fail()
	}

	item3.Next = item4
	queue.Push(item3)

	if queue.First != item2 || queue.Last != item3 {
		test.Log("Queue first / last not set correctly (8)")
		test.Fail()
	}

	if item2.Previous != nil || item3.Next != nil || item3.Previous != item2 || item2.Next != item3 || item3.Queue != queue {
		test.Log("Item previous, next or queue not set correctly (8)")
		test.Fail()
	}

	popped = queue.Pop()
	if popped != item2 {
		test.Log("Wrong popped")
		test.Fail()
	}

	popped = queue.Pop()
	if popped != item3 {
		test.Log("Wrong popped")
		test.Fail()
	}

	if !queue.IsEmpty() || queue.First != nil || queue.Last != nil {
		test.Log("Queue not reset on last pop")
		test.Fail()
	}
}
