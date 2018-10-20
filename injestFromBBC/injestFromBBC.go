package injestFromBBC

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
)

// Taken from https://www.bbc.co.uk/podcasts.json
// Used https://mholt.github.io/json-to-go/
type PodcastsObj struct {
	Podcasts []struct {
		Title           string   `json:"title"`
		ShortTitle      string   `json:"shortTitle"`
		Description     string   `json:"description"`
		NetworkID       string   `json:"networkId"`
		IonServiceID    string   `json:"ionServiceId"`
		LaunchDate      string   `json:"launchDate"`
		LastPublishDate string   `json:"lastPublishDate"`
		Frequency       string   `json:"frequency"`
		LiveItems       int      `json:"liveItems"`
		ImageURL        string   `json:"imageUrl"`
		HomepageURL     string   `json:"homepageUrl"`
		FeedURL         string   `json:"feedUrl"`
		BrandPids       []string `json:"brandPids"`
		Genres          []string `json:"genres"`
	} `json:"podcasts"`
}

func CrawlBBC() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make a request to BBC's podcasts.json to fetch a list of all the podcasts then injest the RSS feed from each one
	resp, err := http.Get("https://www.bbc.co.uk/podcasts.json")
	if err != nil {
		log.Fatal("Error fetching the podcasts.json from BBC")
	}
	defer resp.Body.Close()
	// try to read body into a variable
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading the response body from https://www.bbc.co.uk/podcasts.json")
	}

	var podcastResult = PodcastsObj{}
	err = json.Unmarshal(body, &podcastResult)
	if err != nil {
		log.Printf("%s", err)
		log.Fatal("Unable to unmarshal JSON from body of https://www.bbc.co.uk/podcasts.json")
	}

	for _, podcast := range podcastResult.Podcasts {
		injest.Injest(podcast.FeedURL)
	}

}
