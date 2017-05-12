package crawler

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestWorkerDepth(test *testing.T) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})
	worker1 := NewHostworker("test.com", crawler)

	u, err := url.Parse("http://www.test.com")
	if err != nil {
		test.Fatal(err)
		return
	}
	strBefore := u.Path

	uCopy := *u
	root, err := worker1.NewReference(&uCopy, nil, false)
	if err != nil {
		test.Fatal(err)
		return
	}

	if root.URL.Path != strBefore || uCopy.Path != strBefore {
		test.Log("NewReference changed url")
		test.Log(strBefore)
		test.Log(root.URL.String())
		test.Fail()
	}

	if root.Depth != 0 || root.Queue != worker1.PriorityQueue {
		test.Log("Introduction point depth not set correctly / not on priority")
		test.Fail()
	}

	// Wat als we het nog een keer vinden?

	uCopy = *u
	root2, _ := worker1.NewReference(&uCopy, nil, false)
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
	url1, _ := u.Parse("/contact/")
	url2, _ := u.Parse("/info/")

	// Nieuwe pagina's

	recrawl := 0
	var contact_item *CrawlItem
	for recrawl < 50000 {
		//test.Logf("Recrawl %v\n", recrawl)
		//fmt.Println("Recrawl loop")

		i := 0
		for next != nil {
			worker1.RequestStarted(next)
			originalNext := next.URL.String()

			//fmt.Printf("Depth %v, cycle %v\n", next.Depth, next.Cycle)
			uCopy1 := *u
			root_copy, _ := worker1.NewReference(&uCopy1, next, true)

			if next != root || recrawl == 0 {
				uCopy2 := *url1
				contact_item, _ = worker1.NewReference(&uCopy2, next, true)
			}

			uCopy3 := *url2
			info_item, _ := worker1.NewReference(&uCopy3, next, true)

			if root_copy != root || root_copy.Depth != 0 {
				test.Log("Depth / queue not set correctly for root")
				test.Fail()
			}

			if i == 0 {
				if recrawl == 0 && (contact_item.Depth != 1 || contact_item.Queue != worker1.PriorityQueue) {
					test.Log("Depth / queue not set correctly on repeating pages")
					test.Logf("Contact_item depth = %v\n", contact_item.Depth)

					test.Logf("Contact_item queue = %s\n", contact_item.Queue)
					test.Fail()
				}

				if info_item.Depth != 1 || info_item.Queue != worker1.PriorityQueue {
					test.Log("Depth / queue not set correctly on repeating pages (info_item)")
					test.Logf("info_item depth = %v\n", contact_item.Depth)

					test.Logf("info_item queue = %s\n", contact_item.Queue)
					test.Fail()
				}

				if info_item.Cycle != recrawl {
					test.Log("Cycle not set correctly on repeating pages")
					test.Fail()
				}
			} else {
				if recrawl > 0 {
					if info_item.Depth != 1 {
						test.Log("Depth not set correctly on repeating pages")
						test.Logf("info_item depth = %v\n", info_item.Depth)
						test.Fail()
					}
				} else {
					if contact_item.Depth != 1 || info_item.Depth != 1 {
						test.Log("Depth not set correctly on repeating pages")
						test.Logf("Contact_item depth = %v\n", contact_item.Depth)
						test.Fail()
					}
				}

			}

			if originalNext != next.URL.String() {
				test.Logf("Next url did change from %v, to %v\n", originalNext, next.URL.String())
				test.Fail()
			}

			if next.Depth < 15 {
				url_page1, _ := url.Parse("page1/")
				url_page2, _ := url.Parse("page2/")

				uu := u.ResolveReference(next.URL)

				url_page1_abs := uu.ResolveReference(url_page1)
				url_page2_abs := uu.ResolveReference(url_page2)

				page1, _ := worker1.NewReference(url_page1_abs, next, true)
				page2, _ := worker1.NewReference(url_page2_abs, next, true)

				if page1.Depth != next.Depth+1 || page2.Depth != next.Depth+1 {
					test.Logf("Depth not set correctly on new pages, is %v, should be %v\n", page1.Depth, next.Depth+1)
					test.Log(page1.URL.String())

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

		if recrawl > 2 {
			// Eerste recrawl achter de rug, depth van contact pagina moet aangepast zijn
			if contact_item.Depth != 2 {
				test.Log("Depth for death pages not recovered")
				test.Logf("Contact_item depth = %v != 2\n", contact_item.Depth)
				test.Fail()
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

	//fmt.Printf("Depth %v, cycle %v\n", next.Depth, next.Cycle)
	urlInvalid, _ := u.Parse("/invalid_url/")

	invalidItem, _ := worker1.NewReference(urlInvalid, nil, false)
	invalidItem.Remove()
	worker1.RequestStarted(invalidItem)
	worker1.RequestFailed(invalidItem)

	invalidItem.Remove()
	worker1.RequestStarted(invalidItem)
	worker1.RequestFailed(invalidItem)

	invalidItem.Remove()
	worker1.RequestStarted(invalidItem)
	worker1.RequestFailed(invalidItem)

	invalidItem.Remove()
	worker1.RequestStarted(invalidItem)
	worker1.RequestFailed(invalidItem)

	urlInvalid2, _ := u.Parse("/invalid_url2/")

	invalidItem2, _ := worker1.NewReference(urlInvalid2, nil, false)
	invalidItem2.Remove()
	worker1.RequestStarted(invalidItem2)
	worker1.RequestFailed(invalidItem2)

	invalidItem2.Remove()
	worker1.RequestStarted(invalidItem2)
	worker1.RequestFailed(invalidItem2)

	urlInvalid3, _ := u.Parse("/invalid_url3/")
	invalidItem3, _ := worker1.NewReference(urlInvalid3, nil, false)
	invalidItem3.Remove()
	worker1.RequestStarted(invalidItem3)
	worker1.RequestFailed(invalidItem3)

	// todo: test deze fails

	// We veranderen het nu zodanig dat de contact pagian
	// niet meer bereikbaar is vanaf de startpagina, maar wel nog
	// steeds vanaf de andere pagina's. Hierdoor moet de diepte van die
	// pagina aangepast worden naar 2
	buffer := bytes.NewBufferString("")
	w := bufio.NewWriter(buffer)
	worker1.SaveToWriter(w)
	w.Flush()
	str := buffer.String()

	workerCopy := NewHostworker("", crawler)
	workerCopy.ReadFromReader(bufio.NewReader(strings.NewReader(str)))

	if !worker1.IsEqual(workerCopy) {
		test.Log("Saving worker failed")
		test.Fail()
	}
	worker1.SaveToFile()

	fmt.Printf("%v KB saved", len(str)/1024)
}

// Duurt 1.2 seconden!
func BenchmarkLoadingFromFile(b *testing.B) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})

	for n := 0; n < b.N; n++ {
		file, err := os.Open("./progress/host_test.com.txt")
		if err != nil {
			continue
		}

		NewHostWorkerFromFile(file, crawler)
		file.Close()
	}
}
