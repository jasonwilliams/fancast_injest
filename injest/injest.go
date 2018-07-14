package injest

import (
	"log"
	"regexp"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
)

// For performance, compile this once at the beginning
var (
	UUIDRegex = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	url       string
)

func Injest(input chan string, status chan int) {
	// make sure we're fetching the correct URL, if there's been a 301, this will use the new endpoint
	// This function will also update the DB if there has been a redirect
	for url := range input {
		// checkPodcastUrl can fail if the url is down or 500s
		// lookahead to get metadata, such as headers, redirects etc
		url, NotModified, err := checkPodcastUrl(url)
		if err != nil {
			log.Printf("%s", err)
			continue
		}

		if NotModified {
			log.Println("Request 304 Not Modified, continue")
			continue
		}

		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(url)
		if err != nil {
			log.Printf("Injest: Error parsing %s\n", url)
			log.Println(err)
			continue
		}

		process(feed, url)

	}

	// Signal that we have finished
	status <- 1
}
