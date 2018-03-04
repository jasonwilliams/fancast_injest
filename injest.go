package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

// For performance, compile this once at the beginning
var UUIDRegex = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func main() {
	// Setup Config
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   //
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// make sure we're fetching the correct URL, if there's been a 301, this will use the new endpoint
	url := fetchConanicalUrl("http://podcasts.files.bbci.co.uk/b00lvdrj.rss")
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}

	process(feed, url)
}

// This is purely to make sure we're hitting the most up-to-date URL
// We don't follow any redirects and check the response object to see if its a 301
// if it is then we will make a note of the new endpoint
func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	return errors.New("Don't redirect")
}

func fetchConanicalUrl(feed string) string {
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	resp, err := client.Get(feed)
	if err != nil {
		log.Println("error fetching feed")
		log.Fatal(err)
	}
	location, err := resp.Location()
	if err != nil {
		return feed
	}

	log.Printf("There has been a redirect: %s is now %s\n", feed, location)
	return location.String()

}
