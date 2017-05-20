package main

import (
	"fmt"
	"github.com/SimonBackx/lantern-crawler/crawler"
	"net/url"
)

func run(quit chan bool, finished chan bool) {
	defer func() {
		finished <- true
	}()

	conf := crawler.ConfigFromFile()
	conf.Describe()

	myCrawler := crawler.NewCrawler(conf)

	var urls []string
	if conf.OnlyOnion {
		urls = []string{
			"http://torlinkbgs6aabns.onion/",
			"http://zqktlwi4fecvo6ri.onion/wiki/index.php/Main_Page",
			"http://w363zoq3ylux5rf5.onion/",
			"http://qzbkwswfv5k2oj5d.onion/",
			"http://acropolhwczbgbkh.onion/",
			"http://rhe4faeuhjs4ldc5.onion/",
			"http://s6cco2jylmxqcdeh.onion/",
			"http://s6cco2jylmxqcdeh.onion/",
			"http://eg63fcmp7l7t4vzj.onion/",
			"http://csmania3ljzhig4p.onion/",
			"http://destinysk4bhghnd.onion/",
			"http://hackslciome4eshp.onion/",
			"http://skgmctqnhyvfava3.onion/",
			"http://ogatl57cbva6tncg.onion/",
			"http://flibustahezeous3.onion/",
			"http://aet7lmoi4advnqhy.onion/",
			"http://zeroerfjaacldxzf.onion/",
			"http://hackcanl2o4lvmnv.onion/",
			"http://answerstedhctbek.onion/",
			"http://fcnwebggxt2d3h64.onion/",
		}
	} else {
		urls = []string{"http://www.startpagina.nl"}
	}

	for _, str := range urls {
		u, err := url.Parse(str)
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
