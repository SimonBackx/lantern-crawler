package main

import (
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/crawler"
	//"github.com/SimonBackx/master-project/parser"
	"net/url"
	//"regexp"
)

func run(quit chan bool, finished chan bool) {
	defer func() {
		finished <- true
	}()
	// Website configuratie ophalen
	/*website := &crawler.Website{
		Name:          "Hansa Market",
		URL:           "hansamkt2rr6nfg3.onion",
		ListingRegexp: regexp.MustCompile("/listing/[0-9]+/?"),
		ListingConfiguration: parser.NewListingConfiguration(
			".container .row h2",
			".container .row h3 + p",
			".container .row h2 + .row form table a",
			".listing-price strong",
		),
	}*/

	/*website := &crawler.Website{
	    Name:          "0day.today",
	    URL:           "0day.today",
	    MaxRequests:   1,
	    ListingRegexp: regexp.MustCompile("/exploit/description/[0-9]+/?"),
	    ListingConfiguration: parser.NewListingConfiguration(
	        ".exploit_title h1",                                                // title
	        ".exploit_view_table_content div.td:contains('Description') + .td", // description
	        "div.td:contains('Author') + .td a",                                // author
	        "div.td:contains('Price') + .td .GoldText",                         // price
	    ),
	}*/

	// Door tor sturen
	proxyAddr := "127.0.0.1:9150"
	conf := &config.CrawlerConfig{TorProxyAddress: &proxyAddr}

	// Niet door tor sturen
	//conf := &config.CrawlerConfig{}

	myCrawler := crawler.NewCrawler(conf)

	urls := []string{"http://torlinkbgs6aabns.onion/", "http://zqktlwi4fecvo6ri.onion/wiki/index.php/Main_Page", "http://w363zoq3ylux5rf5.onion/"}

	for _, str := range urls {
		u, err := url.ParseRequestURI(str)
		if err == nil {
			myCrawler.ProcessUrl(u)
		} else {
			fmt.Println(err)
		}
	}

	signal := make(chan int, 1)

	go func() {
		<-quit
		fmt.Println("Sending shutdown signal")
		// Stop signaal sturen naar onze crawler
		signal <- 1
	}()

	myCrawler.Start(signal)
}
