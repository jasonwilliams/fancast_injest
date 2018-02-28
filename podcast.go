package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
	"github.com/mitchellh/hashstructure"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
)

var (
	db      *sql.DB
	connErr error
)

// ProcessPodcast will take a feed object and start inserting the properties into the database
// It will also need to generate an ID for each podcast aswell
func process(feed *gofeed.Feed, url string) {
	// Connect to the database
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s", viper.Get("database.user"), viper.Get("database.database"), viper.Get("database.password"))
	db, connErr = sql.Open("postgres", connStr)
	if connErr != nil {
		log.Fatal(connErr)
	}

	// Does the podcast already exist?
	if podcastExists(url) {
		// Podcast exists, but some data may need updating
	} else {
		id := createNewPodcast(feed, url)
		// createNewPodcastEpisodes(feed, id)
	}

}

// createNewPodcastEpisodes will loop through each episode and add/update the database
func createNewPodcastEpisodes(feed *gofeed.Feed, id string) {
	for i, episode := range feed.Items {
		processPodcastEpisode(episode)
	}
}

// There are 3 states we need to work out...
// Podcast may exist and we don't need to do anything
// Podcast may exist but some metadata is outdated
// Podcast does not exist
func processPodcastEpisode(episode *gofeed.Item) {
	if digestExists(episode) {
		// no need to do anything, this episode is already in the DB and is up to date
	} else if episodeGuidExists(episode) {
		// Episode exists but digest is out of date, add all fields back in
		updateEpisodeInDatabase(episode)
	}
}

func updateEpisodeInDatabase(episode *gofeed.Item) {
	// Generate data
	id := generateIDForPodcast(episode.GUID)
	author, err := json.Marshal(feed.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse author into JSON")
	}

	image, err := json.Marshal(feed.Image)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse image into JSON")
	}

	itunesExt, err := json.Marshal(feed.ITunesExt)
	if err != nil {
		log.Println(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Couldn't begin database transaction")
	}
	_, writeErr := tx.Exec("UPDATE podcast_episodes SET (guid, title, description, published, published_parsed, author, image, itunes_ext, parent) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);",
		episode.GUID, feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, author, feed.Language, image, itunesExt, feed.Copyright, url)
	if writeErr != nil {
		log.Println("Could not write to DB")
		log.Println(writeErr)

		// check if the problem is duplicate ID, this is highly unlikely
		if strings.HasPrefix(writeErr.Error(), "pq: duplicate key value violates unique constraint") {
			log.Println("Duplicate ID generated, trying again....")
			process(feed, url)
			return
		}
	} else {
		commitErr := tx.Commit()
		if commitErr != nil {
			log.Println("Commit failed")
			log.Fatal(commitErr)
		}
	}
}

// digestExists is mainly used by podcast episode objects
// Its a faster way than checking every single property
func episodeGuidExists(episode *gofeed.Item) bool {
	// we don't actually use title here, but it Scan returns an error object which we want
	var title string
	err := db.QueryRow("SELECT title FROM podcast_episodes WHERE guid = $1;", episode.GUID).Scan(&title)
	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatal(err)
	default:
		return true
	}
}

// digestExists is mainly used by podcast episode objects
// Its a faster way than checking every single property
func digestExists(episode *gofeed.Item) bool {
	// lets start by hashing the episode object and see if we have something similar in the DB
	hash, hashErr := hashstructure.Hash(episode, nil)
	if hashErr != nil {
		log.Printf("Cannot Hash episode GUID: %s", episode.GUID)
		log.Fatal(hashErr)
	}

	// we don't actually use title here, but it Scan returns an error object which we want
	var title string
	err := db.QueryRow("SELECT title FROM podcast_episodes WHERE digest = $1;", hash).Scan(&title)
	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatal(err)
	default:
		return true
	}
}

func createNewPodcast(feed *gofeed.Feed, url string) string {
	// Generate data
	id := generateNewID()
	author, err := json.Marshal(feed.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse author into JSON")
	}

	image, err := json.Marshal(feed.Image)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse image into JSON")
	}

	itunesExt, err := json.Marshal(feed.ITunesExt)
	if err != nil {
		log.Println(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Couldn't begin database transaction")
	}
	_, writeErr := tx.Exec("INSERT INTO podcasts(id, title, description, link, updated, updated_parsed, author, language, image, itunes_ext, copyright, feed_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);",
		id, feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, author, feed.Language, image, itunesExt, feed.Copyright, url)
	if writeErr != nil {
		log.Println("Could not write to DB")
		log.Println(writeErr)

		// check if the problem is duplicate ID, this is highly unlikely
		if strings.HasPrefix(writeErr.Error(), "pq: duplicate key value violates unique constraint") {
			log.Println("Duplicate ID generated, trying again....")
			process(feed, url)
			return
		}
	} else {
		commitErr := tx.Commit()
		if commitErr != nil {
			log.Println("Commit failed")
			log.Fatal(commitErr)
		}
	}

	return id
}

// podcastExists checks the database to see if a particular podcast already exists.
// We use the URL as a key to check, as at this point we won't know the GUID
func podcastExists(url string) bool {
	// we don't actually use title here, but it Scan returns an error object which we want
	var title string
	err := db.QueryRow("SELECT title FROM podcasts WHERE feed_url = $1;", url).Scan(&title)
	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		log.Fatal(err)
	default:
		return true
	}
	return false
}

// GUID's of podcasts can vary a LOT
// Some podcasts have perfectly good GUIDS, and we should be able to re-use these
// Some podcasts use the URL as their UUID, this isn't a great ID as it could change (https, change of TLD, change of mp3, new domain)
// Some podcasts don't have any GUID at all, in which case we need to generate one
func generateIDForPodcast(guid string) string {
	// This function could potentially be passed empty strings, its unlikely but can happen
	if len(guid) != 0 {
		// We could be dealing with a Valid UUID, in which case we should just re-use it
		if IsValidUUID(guid) {
			return guid
		}

		// unless they already have a GUID, generate a new GUID
		// generating from a string is dangerous, as its likely we could end up with duplicates
		// so don't take the risk and just generate new GUIDS
		id := generateNewID()

		fmt.Println(id)
		return id
	} else {
		return generateNewID()
	}
}

func generateNewID() string {
	return uuid.NewV4().String()
}

// Check if a UUID is valid
func IsValidUUID(uuid string) bool {
	return UUIDRegex.MatchString(uuid)
}
