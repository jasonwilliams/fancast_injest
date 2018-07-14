/*
	checkPodcastUrl will check the URL and see if it needs updating.
	It does this in 2 ways, first we check if there's been a 301 redirect, if there has then we do a
	lookup on the old URL and if there's a match we update it with the new URL.

	The second option is to check for a flag in the metadata to say the feed has been moved
	More info: https://help.apple.com/itc/podcasts_connect/?lang=en#/itca489031e0
*/

package injest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	_ "github.com/lib/pq"
)

var redirectErr *regexp.Regexp = regexp.MustCompile(`Don't redirect`)

// These are the request headers we plan to send
type RequestHeaders struct {
	Etag         string `json:"etag"`
	LastModified string `json:"last-modified"`
	CacheControl string `json:"cache-control"`
}

// checkPodcastUrl checks to see if there is a redirect in the response,
// If there is a redirect it will return the new URL, otherwise it will return the same url passed in.
// If we already have the podcast, it will also update the database with the new URL
// This is to make sure the database eventually updates with the new URL should a podcast move
func checkPodcastUrl(url string) (string, bool, error) {
	isRedirect, newEndpoint, response, err := fetchConanicalUrl(url)
	if err != nil {
		return "", false, err
	}
	if isRedirect {
		log.Printf("There has been a redirect from %s to %s\n", url, newEndpoint)
		if urlExistsInDB(url) {
			log.Println("Old URL exists, updating to new URL before further injest...")
			updatePodcastUrl(url, newEndpoint)
		}

		return newEndpoint, false, nil
	}

	// We should check if there has been a Not Modified response, in which case we can signal we don't need to go any further
	if response.StatusCode == 304 {
		return url, true, nil
	}

	setHeadersInDB(url, response)

	return url, false, nil
}

func updatePodcastUrl(oldUrl string, newUrl string) {
	// The old URL is in the DB we need to perform a swap
	tx, err := db.Begin()
	if err != nil {
		log.Println("updatePodcastUrl: Couldn't begin database transaction")
		log.Fatal(err)
	}
	_, writeErr := tx.Exec("UPDATE podcasts SET feed_url = $1 WHERE feed_url = $2", newUrl, oldUrl)
	if writeErr != nil {
		log.Println("updatePodcastUrl: Could not write to DB")
		log.Fatal(writeErr)
	}
	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("updatePodcastUrl: Commit failed")
		log.Fatal(commitErr)
	}
}

func urlExistsInDB(url string) bool {
	var urlColumn string
	err := db.QueryRow("SELECT feed_url FROM podcasts WHERE feed_url = $1", url).Scan(&urlColumn)
	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatal(err)
	default:
		return true
	}
	// should never get here
	return false
}

// We don't follow any redirects and check the response object to see if its a 301
// if it is then we will make a note of the new endpoint
func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	return errors.New("Don't redirect")
}

// Return false plus the original URL if there has been no redirect
// Return true plus the new URL if there is a redirect
func fetchConanicalUrl(feed string) (bool, string, *http.Response, error) {
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
		Timeout:       10 * time.Second,
	}

	// Get response headers from previous request before requesting
	requestHeaders := getHeadersFromDB(feed)

	// Create request
	request, _ := http.NewRequest("GET", feed, nil)
	// Set headers to save bandwidth
	request.Header.Add("if-modified-since", requestHeaders.LastModified)
	request.Header.Add("if-none-match", requestHeaders.Etag)

	resp, err := client.Do(request)
	if err != nil {
		log.Println("error fetching feed")
		// It could be a redirect....
		if redirectErr.MatchString(err.Error()) {
			return true, resp.Header.Get("Location"), resp, nil
		}
		// Any other errors
		log.Println(err)
		return false, "", resp, err
	}

	location, err := resp.Location()
	if err != nil {
		return false, feed, resp, nil
	}

	return true, location.String(), resp, nil

}

// setHeadersInDB grabs response headers and saves them into the database for each podcast
// This allows us to use them when making subsequent requests
func setHeadersInDB(url string, response *http.Response) {
	headersToSet := make(map[string]string)
	headersToSet["last-modified"] = response.Header.Get("last-modified")
	headersToSet["etag"] = response.Header.Get("etag")
	headersToSet["cache-control"] = response.Header.Get("cache-control")

	// convert to JSON
	jsonString, err := json.Marshal(headersToSet)
	if err != nil {
		log.Println("unable to Marshal headers from " + url)
	}

	// The old URL is in the DB we need to perform a swap
	tx, err := db.Begin()
	if err != nil {
		log.Println("setHeadersInDB: Couldn't begin database transaction")
		log.Fatal(err)
	}

	_, writeErr := tx.Exec("UPDATE podcasts SET response_headers = $1 WHERE feed_url = $2", jsonString, url)
	if writeErr != nil {
		log.Println("setHeadersInDB: Could not write to DB")
		log.Fatal(writeErr)
	}
	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("setHeadersInDB: Commit failed")
		log.Fatal(commitErr)
	}
}

// getHeadersFromDB returns the response headers from the previous request to feed_url
func getHeadersFromDB(url string) RequestHeaders {
	var responseHeaders sql.NullString
	err := db.QueryRow("SELECT response_headers FROM podcasts WHERE feed_url = $1", url).Scan(&responseHeaders)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%s", err)
	}

	var headers = RequestHeaders{}
	if responseHeaders.Valid {
		err = json.Unmarshal([]byte(responseHeaders.String), &headers)
		if err != nil {
			log.Println("error in getHeaders")
			log.Printf("%s", err)
		}
	}

	return headers
}
