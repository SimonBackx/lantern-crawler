package crawler

import (
	"net/url"
	"testing"
)

func TestWorkerDepth(test *testing.T) {
	crawler := NewCrawler(&CrawlerConfig{Testing: true})
	worker1 := NewHostworker("host1", crawler)

	u, err := url.Parse("/root/")
	test.Log(u.String())
	if err != nil {
		test.Fatal(err)
		return
	}

	root, err := worker1.NewReference(u, nil, false)
	test.Log(root.URL.String())
	test.Log(u.String())

	if err != nil {
		test.Fatal(err)
		return
	}

	if root.URL.String() != u.String() {
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

	i := 0
	for next != nil {
		test.Log(next.URL)
		worker1.RequestStarted(next)

		root_copy, _ := worker1.NewReference(u, next, true)
		contact_item, _ := worker1.NewReference(url1, next, true)
		info_item, _ := worker1.NewReference(url2, next, true)

		url_page1, _ := url.Parse("page1/")
		url_page2, _ := url.Parse("page2/")
		url_page1_abs := next.URL.ResolveReference(url_page1)
		url_page2_abs := next.URL.ResolveReference(url_page2)

		page1, _ := worker1.NewReference(url_page1_abs, next, true)
		page2, _ := worker1.NewReference(url_page2_abs, next, true)

		test.Log(url_page1.String())
		test.Log(url_page1_abs.String())

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
		} else {
			if contact_item.Depth != 1 || info_item.Depth != 1 {
				test.Log("Depth not set correctly on repeating pages")
				test.Logf("Contact_item depth = %v\n", contact_item.Depth)
				test.Fail()
			}
		}

		if page1.Depth != next.Depth+1 || page2.Depth != next.Depth+1 {
			test.Logf("Depth not set correctly on new pages, is %v, should be %v\n", page1.Depth, next.Depth+1)

			test.Fail()
		}

		worker1.RequestFinished(next)

		if root.Queue != worker1.IntroductionPoints {
			test.Log("Root not in introduction points")
			test.Fail()
		}

		i++
		if i > 10000 {
			test.Log("Maximum loop reached")
			test.Fail()
			break
		}

		if next.Depth > 7 {
			break
		}

		next = worker1.GetNextRequest()
	}

	/*
		url_list := [][]string{}

		url, _ := url.Parse("/")
		url1, _ := url.Parse("/contact")
		url2, _ := url.Parse("/info")
		url3, _ := url.Parse("/about")

		url1_1, _ := url.Parse("/contact/page1")
		url1_2, _ := url.Parse("/contact/page2")
		url1_3, _ := url.Parse("/contact/page3")

		url2_1, _ := url.Parse("/info/page1")
		url2_2, _ := url.Parse("/info/page2")
		url2_3, _ := url.Parse("/info/page3")

		url4_1, _ := url.Parse("/about/page1")
		url4_2, _ := url.Parse("/about/page2")
		url4_3, _ := url.Parse("/about/page3")

		// Test 1: diepte basic en queues
		worker1.NewReference(foundUrl, depth, source, sourceItem)*/
}

/*
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, client")
    }))*/
