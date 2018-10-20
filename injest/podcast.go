package injest

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	if feed.ITunesExt != nil && feed.ITunesExt.NewFeedURL != "" && feed.ITunesExt.NewFeedURL != url {
		log.Printf("new feed detected from itunes-new-feed-url: %s", feed.ITunesExt.NewFeedURL)

		// If this is an existing podcast the old URL may exist, in which case we want to replace.
		// But this may not always be the case, so we need to check
		// If the old URL exists, then we need to make the change before we progress further...
		// otherwise we will end up creating a new podcast
		if urlExistsInDB(url) {
			updatePodcastUrl(url, feed.ITunesExt.NewFeedURL)
		}

		url = feed.ITunesExt.NewFeedURL
	}
	// if podcast exists we should get an ID back, we can use this for our further queries
	doesPodcastExist, id, digest := podcastExists(url)
	if doesPodcastExist {
		// Podcast exists in the DB, has there been a change? Lets diff the hashed RSS feeds
		// If they match up then there's no need to update anything
		if generateDigestFromPodcast(feed) != digest {
			// Podcast exists, but some data may need updating
			updatePodcastMetadata(feed, url)
			// This gets all the hashes of the episodes
			episodeHashes := getEpisodesHashesFromPodcast(id)
			processPodcastEpisodes(feed, id, episodeHashes)
		} else {
			// Don't need to do anything but update fetch date
			updateFetchForPodcastURL(url)
		}
	} else {
		// Create a new podcast and return the ID so we can create its children
		id := createNewPodcast(feed, url)
		processPodcastEpisodes(feed, id, make([]string, 0))
	}

}

// processPodcastEpisodes will loop through each episode and add/update the database
func processPodcastEpisodes(feed *gofeed.Feed, id string, hashes []string) {
	for _, episode := range feed.Items {
		processPodcastEpisode(episode, id, hashes)
	}
}

// There are 3 states we need to work out...
// Podcast may exist and we don't need to do anything
// Podcast may exist but some metadata is outdated
// Podcast does not exist
func processPodcastEpisode(episode *gofeed.Item, parent string, hashes []string) {
	if digestExists(episode, hashes) {
		// no need to do anything, this episode is already in the DB and is up to date
	} else if episodeGuidExists(episode) {

		log.Printf("guid exists but change detected on %s\n", episode.GUID)
		log.Println("Reinjesting episode....")
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
	hash := generateDigestFromEpisode(episode)
	m["digest"] = []byte(hash)

	// generate timestamp
	t := time.Now()
	m["last_fetch"] = []byte(t.Format(time.RFC3339))

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

	_, writeErr := tx.Exec("INSERT INTO podcast_episodes (id, guid, title, description, published, published_parsed, author, image, enclosures, digest, itunes_ext, last_fetch, parent) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);",
		id, episode.GUID, episode.Title, episode.Description, episode.Published, episode.PublishedParsed, m["author"], m["image"], m["enclosures"], m["digest"], m["itunesExt"], m["last_fetch"], parent)
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
	_, writeErr := tx.Exec("UPDATE podcast_episodes SET (guid, title, description, published, published_parsed, author, image, enclosures, digest, itunes_ext, last_fetch, parent) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) WHERE guid = $1;",
		episode.GUID, episode.Title, episode.Description, episode.Published, episode.PublishedParsed, m["author"], m["image"], m["enclosures"], m["digest"], m["itunesExt"], m["last_fetch"], parent)

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

func generateDigestFromEpisode(episode *gofeed.Item) string {
	hash, _ := structhash.Hash(episode, 1)
	return hash
}

func generateDigestFromPodcast(feed *gofeed.Feed) string {
	hash, _ := structhash.Hash(feed, 1)
	return hash
}

// digestExists is mainly used by podcast episode objects
// Its a faster way than checking every single property
func digestExists(episode *gofeed.Item, hashes []string) bool {
	// Contains tells whether a contains x.
	contains := func(a []string, x string) bool {
		for _, n := range a {
			if x == n {
				return true
			}
		}
		return false
	}
	digest := generateDigestFromEpisode(episode)
	return contains(hashes, digest)
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

	// Generate hash
	hash := generateDigestFromPodcast(feed)
	m["digest"] = []byte(hash)

	// generate timestamp
	t := time.Now()
	lastFetch := t.Format(time.RFC3339)
	m["last_fetch"] = []byte(lastFetch)

	// generate last change, if hashes are different, date should be now()
	// if hashes are the same, date will match what's already in the DB
	m["last_change"] = []byte(getLastChanged(hash, url))

	return m
}

// updateFetchForPodcastURL updates the timestamp for a podcast (by URL)
func updateFetchForPodcastURL(url string) {
	// generate timestamp
	t := time.Now()
	lastFetch := t.Format(time.RFC3339)

	// Start transcation
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
	}
	query := `UPDATE podcasts SET last_fetch = $1 where feed_url = $2`
	_, writeErr := tx.Exec(query, lastFetch, url)
	if writeErr != nil {
		log.Println("updateFetchForURL: Could not write to DB")
		log.Println(writeErr)
	}
	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("updateFetchForURL: Commit failed")
		log.Fatal(commitErr)
	}

}

// Update POLL Frequency
/**
	POLL Frequency - No point polling old podcasts if they're inactive, here's a table
	> 730 hours (1 month) -- 48 hours
	> 168 -- 24 hours
	> 48 -- 16 hours
	> 24 -- 8 hours
	default -- 4 hours
**/
func updatePollFrequency(url string) int8 {
	var (
		lastChange sql.NullString
	)

	err := db.QueryRow("SELECT last_change FROM podcasts WHERE feed_url = $1;", url).Scan(&lastChange)
	if err != nil {
		log.Println(err)
	}
	// Setup time for comparison
	t := time.Now()

	// no last change date, set default time
	if !lastChange.Valid {
		return 4
	}

	// Parse lastChange into time
	lastChangeTime, err := time.Parse(time.RFC3339, lastChange.String)
	if err != nil {
		log.Println(err)
	}

	// get the difference from now to lastChange
	diff := t.Sub(lastChangeTime).Hours()

	if diff > 730 {
		return 48
	} else if diff > 168 {
		return 24
	} else if diff > 48 {
		return 16
	} else if diff > 24 {
		return 8
	} else {
		return 4
	}
}

// This function works out the last time this podcasts had changed (different to last_fetch which records last fetch)
// Then returns a time
// If there is a digest, check if its the same, if so continue to use same last_change date
// If no digest is set, last_change should be now
func getLastChanged(hash, url string) string {
	var (
		digest sql.NullString
		change sql.NullString
	)
	err := db.QueryRow("SELECT digest, last_change FROM podcasts WHERE feed_url = $1;", url).Scan(&digest, &change)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}

		// This is a new podcast, there is no data, just generate current time
		return time.Now().Format(time.RFC3339)
	}

	// If there's no digest, just send the current time
	if !digest.Valid {
		return time.Now().Format(time.RFC3339)
	}

	// If there's no last_change send back current time
	if !change.Valid {
		return time.Now().Format(time.RFC3339)
	}

	// if these match, there has been no change
	if hash == digest.String {
		return change.String
	}

	// If we reach here, then we have a hash and a last_change, but there's been a change
	// Replace last_change with new date
	return time.Now().Format(time.RFC3339)
}

func updatePodcastMetadata(feed *gofeed.Feed, url string) {
	// For all the JSON properties, create a new mapping
	m := preparePodcastForDB(feed)
	freq := updatePollFrequency(url)
	tx, err := db.Begin()
	if err != nil {
		log.Println("updatePodcastMetadata: Couldn't begin database transaction")
		log.Fatal(err)
	}

	query := `
	UPDATE podcasts SET (last_fetch, title, description, link, updated, updated_parsed, author, language, image, itunes_ext, categories, copyright, poll_frequency, last_change, digest) =
	($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) where feed_url = $1;
	`
	_, writeErr := tx.Exec(query, url, m["last_fetch"], feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, m["author"], feed.Language, m["image"], m["ItunesExt"], m["categories"], feed.Copyright, freq, m["last_change"], m["digest"])
	if writeErr != nil {
		log.Println("updatePodcastMetadata: Could not write to DB")
		log.Println(writeErr)
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
	// _, writeErr := tx.Exec("INSERT INTO podcasts(id, title, description, link, updated, updated_parsed, author, language, image, itunes_ext, categories, copyright, last_fetch, feed_url, digest, poll_frequency) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);",
	// id, feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, m["author"], feed.Language, m["image"], m["ItunesExt"], m["categories"], feed.Copyright, m["last_fetch"], url, m["digest"], 8)
	query := `
	INSERT INTO podcasts (id, last_fetch, title, description, link, updated, updated_parsed, author, language, image, itunes_ext, categories, copyright, poll_frequency, last_change, digest, feed_url) VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);
	`
	res, writeErr := tx.Exec(query, id, m["last_fetch"], feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, m["author"], feed.Language, m["image"], m["ItunesExt"], m["categories"], feed.Copyright, 8, m["last_change"], m["digest"], url)
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

	r, _ := res.RowsAffected()
	log.Println(r)

	return id
}

// podcastExists checks the database to see if a particular podcast already exists.
// We use the URL as a key to check, as at this point we won't know the GUID
func podcastExists(url string) (bool, string, string) {
	// we don't actually use title here, but it Scan returns an error object which we want
	var id string
	var digest sql.NullString
	err := db.QueryRow("SELECT id, digest FROM podcasts WHERE feed_url = $1;", url).Scan(&id, &digest)
	switch {
	case err == sql.ErrNoRows:
		return false, "", id
	case err != nil:
		log.Fatal(err)
	default:
		return true, id, digest.String
	}

	// we would never get here
	return true, id, digest.String
}

// getEpisodesHashesFromPodcast gets all of the episode hashes from a single podcast
// This should save us a lot of time (not connecting to the DB for each episode and checking it exists)
func getEpisodesHashesFromPodcast(id string) []string {
	rows, err := db.Query("SELECT digest FROM podcast_episodes WHERE parent = $1;", id)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	digests := make([]string, 0)
	for rows.Next() {
		var digest string
		if err := rows.Scan(&digest); err != nil {
			log.Println(err)
		}
		digests = append(digests, digest)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return digests

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
	}
	return generateNewID()

}

func generateNewID() string {
	return uuid.NewV4().String()
}

// IsValidUUID checks if a UUID is valid
func IsValidUUID(uuid string) bool {
	return UUIDRegex.MatchString(uuid)
}
