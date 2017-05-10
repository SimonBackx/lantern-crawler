package crawler

import (
	"net/url"
	"testing"
)

func TestWorkerDepth(test *testing.T) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})
	worker1 := NewHostworker("host1", crawler)

	u, err := url.Parse("/root/")
	if err != nil {
		test.Fatal(err)
		return
	}

	strBefore := u.String()

	root, err := worker1.NewReference(u, nil, false)
	if err != nil {
		test.Fatal(err)
		return
	}

	if root.URL.String() != strBefore || u.String() != strBefore {
		test.Log("NewReference changed url")
		test.Fail()
	}

	if root.Depth != 0 || root.Queue != worker1.PriorityQueue {
		test.Log("Introduction point depth not set correctly / not on priority")
		test.Fail()
	}

	// Wat als we het nog een keer vinden?
	root2, _ := worker1.NewReference(u, nil, false)
	if root != root2 || root2.Depth != 0 {
		test.Log("Introduction point duplication / depth not set correctly")
		test.Fail()
	}

	next := worker1.GetNextRequest()
	if next != root {
		test.Log("Introduction point not in GetNextRequest")
		test.Fail()
	}

	// Deze urls vinden we terug op elke pagina (samen met root)
	url1, _ := url.Parse("/contact/")
	url2, _ := url.Parse("/info/")

	// Nieuwe pagina's

	recrawl := 0
	for recrawl < 50000 {
		test.Logf("Recrawl %v\n", recrawl)
		//fmt.Println("Recrawl loop")

		i := 0
		for next != nil {
			worker1.RequestStarted(next)

			//fmt.Printf("Depth %v, cycle %v\n", next.Depth, next.Cycle)

			root_copy, _ := worker1.NewReference(u, next, true)
			contact_item, _ := worker1.NewReference(url1, next, true)
			info_item, _ := worker1.NewReference(url2, next, true)

			if root_copy != root || root_copy.Depth != 0 {
				test.Log("Depth / queue not set correctly for root")
				test.Fail()
			}

			if i == 0 {
				if contact_item.Depth != 1 || contact_item.Queue != worker1.PriorityQueue || info_item.Depth != 1 || info_item.Queue != worker1.PriorityQueue {
					test.Log("Depth / queue not set correctly on repeating pages")
					test.Logf("Contact_item depth = %v\n", contact_item.Depth)

					test.Logf("Contact_item queue = %s\n", contact_item.Queue)
					test.Fail()
				}

				if contact_item.Cycle != recrawl {
					test.Log("Cycle not set correctly on repeating pages")
					test.Fail()
				}
			} else {
				if contact_item.Depth != 1 || info_item.Depth != 1 {
					test.Log("Depth not set correctly on repeating pages")
					test.Logf("Contact_item depth = %v\n", contact_item.Depth)
					test.Fail()
				}
			}

			if next.Depth < 15 {
				url_page1, _ := url.Parse("page1/")
				url_page2, _ := url.Parse("page2/")
				url_page1_abs := next.URL.ResolveReference(url_page1)
				url_page2_abs := next.URL.ResolveReference(url_page2)

				page1, _ := worker1.NewReference(url_page1_abs, next, true)
				page2, _ := worker1.NewReference(url_page2_abs, next, true)

				if page1.Depth != next.Depth+1 || page2.Depth != next.Depth+1 {
					test.Logf("Depth not set correctly on new pages, is %v, should be %v\n", page1.Depth, next.Depth+1)

					test.Fail()
				}

				if page1.Depth >= maxRecrawlDepth && ((page1.Queue != worker1.Queue && page1.Cycle == 0) || (page1.Queue != worker1.LowPriorityQueue && page1.Cycle > 0)) {
					test.Log("Queue not set correctly for page1")
					test.Logf("page1 queue = %s\n", page1.Queue)
					test.Fail()
				}

				if page2.Depth >= maxRecrawlDepth && ((page2.Queue != worker1.Queue && page2.Cycle == 0) || (page2.Queue != worker1.LowPriorityQueue && page2.Cycle > 0)) {
					test.Log("Queue not set correctly for page2")
					test.Fail()
				}

				if page1.Depth < maxRecrawlDepth && page1.Queue != worker1.PriorityQueue {
					test.Log("Queue not set correctly for page1")
					test.Fail()
				}

				if page2.Depth < maxRecrawlDepth && page2.Queue != worker1.PriorityQueue {
					test.Log("Queue not set correctly for page2")
					test.Fail()
				}
			}

			worker1.RequestFinished(next)
			if root.Queue != worker1.IntroductionPoints {
				test.Log("Root not in introduction points")
				test.Fail()
			}

			i++
			if i > 80 {
				recrawl++
				worker1.Recrawl()
				if root.Queue != worker1.PriorityQueue || root.Cycle != recrawl {
					test.Log("Recrawl failed")
					test.Fail()
				}

				// Clean worker introduction
				select {
				case <-crawler.WorkerIntroduction:
				default:
					test.Log("WorkerIntroduction not set")
					test.FailNow()
				}
				next = worker1.GetNextRequest()
				break
			}
			next = worker1.GetNextRequest()

			if test.Failed() {
				test.Fatal("infinite loop?")
			}
		}

		if worker1.Queue.IsEmpty() {
			break
		} else {
			if next == nil {
				test.Log("New items but next is nil")
				test.Fail()
				break
			}
		}
	}

	// We veranderen het nu zodanig dat de contact pagian
	// niet meer bereikbaar is vanaf de startpagina, maar wel nog
	// steeds vanaf de andere pagina's. Hierdoor moet de diepte van die
	// pagina aangepast worden naar 2

}
