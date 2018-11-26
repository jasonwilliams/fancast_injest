package injest

import (
	"regexp"

	// Prelude for sql package
	"bitbucket.org/jayflux/mypodcasts_injest/logger"
	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
)

// For performance, compile this once at the beginning
var (
	UUIDRegex = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	url       string
	log       = logger.Log
)

func Injest(url string) {
	// checkPodcastUrl can fail if the url is down or 500s
	// lookahead to get metadata, such as headers, redirects etc
	url, NotModified, err := checkPodcastUrl(url)
	if err != nil {
		log.Printf("%s", err)
		return
	}

	if NotModified {
		log.Printf("Request 304 Not Modified for %s", url)
		// Even though we got a not modified response we should still record a fetch has happened
		updateFetchForPodcastURL(url)
		return
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Printf("Injest: Error parsing %s\n", url)
		log.Println(err)
		// Early return instead of fatal erroring, hopefully this should keep the process running
		return
	}

	process(feed, url)

}
