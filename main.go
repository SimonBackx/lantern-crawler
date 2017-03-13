package main

import (
	"fmt"
	"github.com/SimonBackx/master-project/crawler"
	"github.com/SimonBackx/master-project/parser"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os" // bestand lezen
	"regexp"
)

func main() {
	fmt.Println("Visstok")

	files, _ := ioutil.ReadDir("./data/precrawled/")
	for _, f := range files {

		// Bestand openen met read access
		file, err := os.Open("data/precrawled/" + f.Name())

		if err != nil {
			log.Fatal(err)
		}

		crawler.InitialiseWebsites()
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
		)

		crawler.AddWebsite(
			&crawler.Website{
				Name:          "0day.today",
				URL:           "0day.today",
				ListingRegexp: regexp.MustCompile("/exploit/description/[0-9]+/?"),
				ListingConfiguration: parser.NewListingConfiguration(
					".exploit_title h1",                                                // title
					".exploit_view_table_content div.td:contains('Description') + .td", // description
					"div.td:contains('Author') + .td a",                                // author
					"div.td:contains('Price') + .td .GoldText",                         // price
				),
			},
		)

		urlString := filenameToUrl(f.Name())
		fmt.Println(urlString)

		url, err2 := url.Parse(urlString)
		if err2 != nil {
			fmt.Println(err2.Error())
		} else {
			parseUrl(url, file)
		}
	}

}

func filenameToUrl(filename string) string {
	fileNameMap := make(map[string]string, 0)
	fileNameMap["^hansa_([0-9]+).tor_html$"] = "http://hansamkt2rr6nfg3.onion/listing/$1/"
	fileNameMap["^0day_([0-9]+).tor_html$"] = "http://0day.today/exploit/description/$1"

	for k, v := range fileNameMap {
		reg := regexp.MustCompile(k)
		filename = reg.ReplaceAllString(filename, v)
	}
	return filename
}

func parseUrl(url *url.URL, reader io.Reader) *parser.ParseResult {
	// Website zoeken
	web := crawler.GetWebsiteForDomain(url.Hostname())
	if web == nil {
		fmt.Println("Website", url.Hostname(), "not supported\n")
		return nil
	}

	result, err := parser.Parse(reader, web.GetParsers(url))
	if err != nil || result == nil {
		fmt.Println(err.Error())
		fmt.Println("")
		return nil
	}
	if result.Listing != nil {
		result.Listing.Print()
	} else {
		fmt.Println("No listing found")
	}
	fmt.Println("")
	return result
}
