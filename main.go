package main

import (
	"fmt"
	"github.com/SimonBackx/master-project/config"
	"github.com/SimonBackx/master-project/crawler"
	//"github.com/SimonBackx/master-project/parser"
	"net/url"
	//"regexp"
)

func main() {
	// Website configuratie ophalen
	/*crawler.InitialiseWebsites()
	crawler.AddWebsite(
		&crawler.Website{
			Name:          "Hansa Market",
			URL:           "hansamkt2rr6nfg3.onion",
			ListingRegexp: regexp.MustCompile("/listing/[0-9]+/?"),
			ListingConfiguration: parser.NewListingConfiguration(
				".container .row h2",
				".container .row h3 + p",
				".container .row h2 + .row form table a",
				".listing-price strong",
			),
		},
	)*/

	/*website := &crawler.Website{
		Name:          "0day.today",
		URL:           "0day.today",
		ListingRegexp: regexp.MustCompile("/exploit/description/[0-9]+/?"),
		ListingConfiguration: parser.NewListingConfiguration(
			".exploit_title h1",                                                // title
			".exploit_view_table_content div.td:contains('Description') + .td", // description
			"div.td:contains('Author') + .td a",                                // author
			"div.td:contains('Price') + .td .GoldText",                         // price
		),
	}*/

	website := &crawler.Website{URL: "www.facebookcorewwwi.onion"}
	myCrawler := crawler.NewCrawler(&config.CrawlerConfig{TorProxyAddress: "127.0.0.1:9150"})

	//u, err := url.ParseRequestURI("http://hansamkt2rr6nfg3.onion/listing/10269/")
	u, err := url.ParseRequestURI("https://www.facebookcorewwwi.onion")
	if err == nil {
		myCrawler.AddDomain(crawler.NewDomainCrawler(website))

		fmt.Println("Crawl started")
		myCrawler.ProcessUrl(u)
		myCrawler.Start()
	} else {
		fmt.Println(err)
	}
}
