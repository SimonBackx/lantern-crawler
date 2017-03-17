package main

import (
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/crawler"
	"github.com/SimonBackx/master-project/parser"
	"net/url"
	"regexp"
)

func run(quit chan bool, finished chan bool) {
	defer func() {
		finished <- true
	}()
	// Website configuratie ophalen
	website := &crawler.Website{
		Name:          "Hansa Market",
		URL:           "hansamkt2rr6nfg3.onion",
		MaxRequests:   1,
		ListingRegexp: regexp.MustCompile("/listing/[0-9]+/?"),
		ListingConfiguration: parser.NewListingConfiguration(
			".container .row h2",
			".container .row h3 + p",
			".container .row h2 + .row form table a",
			".listing-price strong",
		),
	}

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

	//website := &crawler.Website{URL: "www.scoutswetteren.be", MaxRequests: 10}

	// Door tor sturen
	proxyAddr := "127.0.0.1:9150"
	conf := &config.CrawlerConfig{TorProxyAddress: &proxyAddr}

	// Niet door tor sturen
	//conf := &config.CrawlerConfig{}

	myCrawler := crawler.NewCrawler(conf)

	u, err := url.ParseRequestURI("http://hansamkt2rr6nfg3.onion")
	//u, err := url.ParseRequestURI("https://www.scoutswetteren.be")
	if err == nil {
		myCrawler.AddDomain(crawler.NewDomainCrawler(website))

		myCrawler.ProcessUrl(u)
		signal := make(chan int, 1)

		go func() {
			<-quit
			fmt.Println("Sending shutdown signal")
			// Stop signaal sturen naar onze crawler
			signal <- 1
			myCrawler.Wake()
		}()

		myCrawler.Start(signal)

		// Crawler is gestopt
	} else {
		fmt.Println(err)
	}
}
