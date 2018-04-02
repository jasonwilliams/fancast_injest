package injest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cnf/structhash"
	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
)

var (
	db      *sql.DB
	connErr error
)

func init() {
	// Setup Viper Config
	setupConfig()

	// Connect to the database
	log.Println("Connecting to Database..")
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s", viper.Get("database.user"), viper.Get("database.database"), viper.Get("database.password"))
	db, connErr = sql.Open("postgres", connStr)
	if connErr != nil {
		log.Fatal(connErr)
	}
}

func setupConfig() {
	// Setup Config
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   //
	viper.AddConfigPath(".")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.database", "DB_NAME")
	viper.BindEnv("database.password", "DB_PASS")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

// ProcessPodcast will take a feed object and start inserting the properties into the database
// It will also need to generate an ID for each podcast aswell
func process(feed *gofeed.Feed, url string) {
	// Does the podcast already exist?
	var doesPodcastExist bool
	var id string

	// Is there a new-feed element? And is it set to the same URL? (BBC ones seem to point to the same URL)
	if feed.ITunesExt.NewFeedURL != "" && feed.ITunesExt.NewFeedURL != url {
		log.Printf("new feed detected from itunes-new-feed-url: %s", feed.ITunesExt.NewFeedURL)

		// If this is an existing podcast the old URL may exist, in which case we want to replace.
		// But this may not always be the case, so we need to check
		// If the old URL exists, then we need to make the change before we progress further...
		// otherwise we will end up creating a new podcast
		if urlExistsInDB(url) {
			log.Println("Updating DB")
			updatePodcastUrl(url, feed.ITunesExt.NewFeedURL)
		}

		url = feed.ITunesExt.NewFeedURL
	}
	// if podcast exists we should get an ID back, we can use this for our further queries
	doesPodcastExist, id = podcastExists(url)
	if doesPodcastExist {
		// Podcast exists, but some data may need updating
		updatePodcastMetadata(feed, url)
		processPodcastEpisodes(feed, id)
	} else {
		// Create a new podcast and return the ID so we can create its children
		id := createNewPodcast(feed, url)
		processPodcastEpisodes(feed, id)
	}

}

// processPodcastEpisodes will loop through each episode and add/update the database
func processPodcastEpisodes(feed *gofeed.Feed, id string) {
	for _, episode := range feed.Items {
		processPodcastEpisode(episode, id)
	}
}

// There are 3 states we need to work out...
// Podcast may exist and we don't need to do anything
// Podcast may exist but some metadata is outdated
// Podcast does not exist
func processPodcastEpisode(episode *gofeed.Item, parent string) {
	if digestExists(episode) {
		// no need to do anything, this episode is already in the DB and is up to date
	} else if episodeGuidExists(episode) {

		log.Printf("guid exists but change detected on %s\n", episode.GUID)
		log.Println("Reinjesting....")
		// Episode exists but digest is out of date, add all fields back in
		updateEpisodeInDatabase(episode, parent)
	} else {
		addEpisodeInDatabase(episode, parent)
	}
}

func prepareEpisodeForDB(episode *gofeed.Item) map[string][]byte {
	var err error
	m := make(map[string][]byte)

	m["author"], err = json.Marshal(episode.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse author into JSON")
	}

	m["image"], err = json.Marshal(episode.Image)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse image into JSON")
	}

	m["itunesExt"], err = json.Marshal(episode.ITunesExt)
	if err != nil {
		log.Println(err)
	}

	m["enclosures"], err = json.Marshal(episode.Enclosures)
	if err != nil {
		log.Println(err)
	}

	// Generate hash
	hash, err := generateDigestFromEpisode(episode)
	m["digest"] = []byte(hash)
	if err != nil {
		log.Fatal(err)
	}

	// generate timestamp
	t := time.Now()
	m["last_processed"] = []byte(t.Format(time.RFC3339))

	return m
}

func addEpisodeInDatabase(episode *gofeed.Item, parent string) {
	// Generate data
	id := generateIDForPodcast(episode.GUID)
	m := prepareEpisodeForDB(episode)

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Couldn't begin database transaction")
	}

	_, writeErr := tx.Exec("INSERT INTO podcast_episodes (id, guid, title, description, published, published_parsed, author, image, enclosures, digest, itunes_ext, last_processed, parent) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);",
		id, episode.GUID, episode.Title, episode.Description, episode.Published, episode.PublishedParsed, m["author"], m["image"], m["enclosures"], m["digest"], m["itunesExt"], m["last_processed"], parent)
	if writeErr != nil {
		log.Printf("Could not write episode (GUID: %s) to DB\n", episode.GUID)
		log.Println(writeErr)
	} else {
		commitErr := tx.Commit()
		if commitErr != nil {
			log.Println("Commit failed")
			log.Fatal(commitErr)
		}
	}

}

func updateEpisodeInDatabase(episode *gofeed.Item, parent string) {
	m := prepareEpisodeForDB(episode)

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("updateEpisodeInDatabase: Couldn't begin database transaction")
	}
	_, writeErr := tx.Exec("UPDATE podcast_episodes SET (guid, title, description, published, published_parsed, author, image, enclosures, digest, itunes_ext, last_processed, parent) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) WHERE guid = $1;",
		episode.GUID, episode.Title, episode.Description, episode.Published, episode.PublishedParsed, m["author"], m["image"], m["enclosures"], m["digest"], m["itunesExt"], m["last_processed"], parent)

	if writeErr != nil {
		log.Printf("updateEpisodeInDatabase: Could not write episode (GUID: %s) to DB\n", episode.GUID)
		log.Println(writeErr)
	} else {
		commitErr := tx.Commit()
		if commitErr != nil {
			log.Println("updateEpisodeInDatabase: Commit failed")
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

	return false
}

// TODO: change to sha256
func generateDigestFromEpisode(episode *gofeed.Item) (string, error) {
	hash, hashErr := structhash.Hash(episode, 1)
	return hash, hashErr
}

// digestExists is mainly used by podcast episode objects
// Its a faster way than checking every single property
func digestExists(episode *gofeed.Item) bool {

	// lets start by hashing the episode object and see if we have something similar in the DB
	hash, hashErr := generateDigestFromEpisode(episode)
	if hashErr != nil {
		log.Println("Failed to hash episode guid: %s ", episode.GUID)
		log.Fatal(hashErr)
	}
	// we don't actually use title here, but it Scan returns an error object which we want
	var title string
	var isRows bool
	err := db.QueryRow("SELECT title FROM podcast_episodes WHERE digest = $1;", hash).Scan(&title)
	switch {
	case err == sql.ErrNoRows:
		isRows = false
	case err != nil:
		log.Fatal(err)
	default:
		isRows = true
	}

	return isRows
}

func preparePodcastForDB(feed *gofeed.Feed) map[string][]byte {
	var err error
	m := make(map[string][]byte)
	m["author"], err = json.Marshal(feed.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse author into JSON")
	}

	m["image"], err = json.Marshal(feed.Image)
	if err != nil {
		log.Println(err)
		log.Fatal("could not parse image into JSON")
	}

	m["ItunesExt"], err = json.Marshal(feed.ITunesExt)
	if err != nil {
		log.Println(err)
	}

	m["categories"], err = json.Marshal(feed.Categories)
	if err != nil {
		log.Println(err)
	}
	// generate timestamp
	t := time.Now()
	last_processed := t.Format(time.RFC3339)
	m["last_processed"] = []byte(last_processed)

	return m
}

func updatePodcastMetadata(feed *gofeed.Feed, url string) {
	// For all the JSON properties, create a new mapping
	m := preparePodcastForDB(feed)
	tx, err := db.Begin()
	if err != nil {
		log.Println("updatePodcastMetadata: Couldn't begin database transaction")
		log.Fatal(err)
	}

	_, writeErr := tx.Exec("UPDATE podcasts SET (title, description, link, updated, updated_parsed, author, language, image, itunes_ext, categories, copyright, last_processed, feed_url) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) WHERE feed_url = $13;",
		feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, m["author"], feed.Language, m["image"], m["ItunesExt"], m["categories"], feed.Copyright, m["last_processed"], url)
	if writeErr != nil {
		log.Println("updatePodcastMetadata: Could not write to DB")
		log.Fatal(writeErr)
	}
	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("Commit failed")
		log.Fatal(commitErr)
	}

}

func createNewPodcast(feed *gofeed.Feed, url string) string {
	// Generate data
	id := generateNewID()
	m := preparePodcastForDB(feed)

	tx, err := db.Begin()
	if err != nil {
		log.Println("createNewPodcast: Couldn't begin database transaction")
		log.Fatal(err)
	}
	_, writeErr := tx.Exec("INSERT INTO podcasts(id, title, description, link, updated, updated_parsed, author, language, image, itunes_ext, categories, copyright, last_processed, feed_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);",
		id, feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, m["author"], feed.Language, m["image"], m["ItunesExt"], m["categories"], feed.Copyright, m["last_processed"], url)
	if writeErr != nil {
		log.Println("Could not write to DB")
		log.Println(writeErr)

		// check if the problem is duplicate ID, this is highly unlikely
		if strings.HasPrefix(writeErr.Error(), "pq: duplicate key value violates unique constraint") {
			log.Println("Duplicate ID generated, trying again....")
			process(feed, url)
			return id
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
func podcastExists(url string) (bool, string) {
	// we don't actually use title here, but it Scan returns an error object which we want
	var id string
	err := db.QueryRow("SELECT id FROM podcasts WHERE feed_url = $1;", url).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		return false, id
	case err != nil:
		log.Fatal(err)
	default:
		return true, id
	}

	// we would never get here
	return true, id
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
