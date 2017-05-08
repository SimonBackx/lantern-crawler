package crawler

import (
	"net/url"
	"testing"
)

func TestLeveledQueue(test *testing.T) {
	u, _ := url.Parse("https://domain.test/hello")
	queue := NewLeveledQueue()
	item1 := NewCrawlItem(u)
	item2 := NewCrawlItem(u)
	item3 := NewCrawlItem(u)
	item4 := NewCrawlItem(u)
	item5 := NewCrawlItem(u)
	item6 := NewCrawlItem(u)
	item7 := NewCrawlItem(u)
	item8 := NewCrawlItem(u)
	item9 := NewCrawlItem(u)

	queue.Push(item1, 1)
	queue.Push(item2, 1)
	queue.Push(item3, 1)
	queue.Push(item4, 2)

	queue.Push(item5, 2)
	queue.Push(item6, 2)
	queue.Push(item7, 3)
	queue.Push(item8, 4)
	queue.Push(item9, 4)

	if queue.First() != item1 {
		test.Log("First item not right")
		test.Fail()
	}

	if queue.Pop() != nil {
		test.Log("Pop not empty")
		test.Fail()
	}

	item4.FakeRetry()
	item5.FakeRetry()
	item8.FakeRetry()

	if queue.Pop() != item4 {
		test.Log("Wrong item popped")
		test.Fail()
	}

	if queue.Pop() != item5 {
		test.Log("Wrong item popped")
		test.Fail()
	}

	if queue.Pop() != item8 {
		test.Log("Wrong item popped")
		test.Fail()
	}

	item1.Remove()

	if queue.First() != item2 {
		test.Log("First item not right")
		test.Fail()
	}

	item2.FakeRetry()

	if queue.Pop() != item2 {
		test.Log("Wrong item popped")
		test.Fail()
	}

}
