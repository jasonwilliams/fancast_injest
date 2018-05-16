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
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	_ "github.com/lib/pq"
)

var redirectErr *regexp.Regexp = regexp.MustCompile(`Don't redirect`)

func checkPodcastUrl(url string) (string, error) {
	isRedirect, newEndpoint, err := fetchConanicalUrl(url)
	if err != nil {
		return "", err
	}
	if isRedirect {
		log.Printf("There has been a redirect from %s to %s\n", url, newEndpoint)
		if urlExistsInDB(url) {
			log.Println("Old URL exists, updating to new URL before further injest...")
			updatePodcastUrl(url, newEndpoint)
		}

		return newEndpoint, nil
	}
	return url, nil
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
func fetchConanicalUrl(feed string) (bool, string, error) {
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
		Timeout:       10 * time.Second,
	}

	resp, err := client.Get(feed)
	if err != nil {
		log.Println("error fetching feed")
		// It could be a redirect....
		if redirectErr.MatchString(err.Error()) {
			log.Println("Redirect found... ")
			log.Println(resp.Header.Get("Location"))
			return true, resp.Header.Get("Location"), nil
		}
		// Any other errors
		log.Println(err)
		return false, "", err
	}

	location, err := resp.Location()
	if err != nil {
		return false, feed, nil
	}

	return true, location.String(), nil

}
