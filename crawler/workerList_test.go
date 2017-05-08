package crawler

import (
	"testing"
)

func TestWorkerList(test *testing.T) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})
	worker1 := NewHostworker("host1", crawler)
	worker2 := NewHostworker("host2", crawler)
	worker3 := NewHostworker("host3", crawler)
	worker4 := NewHostworker("host4", crawler)
	worker5 := NewHostworker("host5", crawler)
	worker6 := NewHostworker("host6", crawler)

	list := NewWorkerList()
	list.Push(worker1)

	if list.First.Worker != worker1 || list.Last.Worker != worker1 {
		test.Log("List first / last not set correctly")
		test.Fail()
	}

	list.Push(worker2)

	if list.First.Worker != worker1 || list.Last.Worker != worker2 {
		test.Log("List first / last not set correctly")
		test.Fail()
	}

	list.Push(worker3)
	list.Push(worker4)
	list.Push(worker5)

	if list.First.Worker != worker1 || list.Last.Worker != worker5 {
		test.Log("List first / last not set correctly")
		test.Fail()
	}

	popped := list.Pop()

	if popped != worker1 {
		test.Log("Popped wrong")
		test.Fail()
	}

	if list.First.Worker != worker2 || list.Last.Worker != worker5 {
		test.Log("List first / last not set correctly")
		test.Fail()
	}
	popped = list.Pop()

	if popped != worker2 {
		test.Log("Popped wrong")
		test.Fail()
	}

	if list.First.Worker != worker3 || list.Last.Worker != worker5 {
		test.Log("List first / last not set correctly")
		test.Fail()
	}

	list.Pop()
	list.Pop()
	list.Pop()

	if list.First != nil || list.Last != nil || !list.IsEmpty() {
		test.Log("List first / last not correctly reset")
		test.Fail()
	}

	list.Push(worker5)
	list.Push(worker6)

	if list.First.Worker != worker5 || list.First.Next.Worker != worker6 || list.First.Next.Next != nil {
		test.Log("Next not ok")
		test.Fail()
	}

}
