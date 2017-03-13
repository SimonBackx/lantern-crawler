package crawler

var websiteMap map[string]*Website

func InitialiseWebsites() {
	websiteMap = make(map[string]*Website, 0)
}

func GetWebsiteForDomain(domain string) *Website {
	return websiteMap[domain]
}

func AddWebsite(web *Website) {
	websiteMap[web.URL] = web
}
