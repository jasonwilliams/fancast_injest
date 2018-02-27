package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
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

	// fp := gofeed.NewParser()
	// feed, err := fp.ParseURL("http://atp.fm/episodes?format=rss")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	processPodcast(feed)
	// fetchConanicalUrl("https://bbc.co.uk")
}

// ProcessPodcast will take a feed object and start inserting the properties into the database
// It will also need to generate an ID for each podcast aswell
func processPodcast(feed *gofeed.Feed) {
	// Connect to the database
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s", viper.Get("database.user"), viper.Get("database.database"), viper.Get("database.password"))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

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

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Couldn't begin database transaction")
	}
	_, writeErr := tx.Exec("INSERT INTO podcasts(id, title, description, link, updated, updatedParsed, published, publishedParsed, author, language, image, itunesExt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11. $12);", id, feed.Title, feed.Description, feed.Link, feed.Updated, feed.UpdatedParsed, feed.Published, feed.PublishedParsed, author, feed.Language, image, feed.ITunesExt)
	if writeErr != nil {
		log.Println("Could not write to DB")
		log.Println(writeErr)

		// check if the problem is duplicate ID, this is highly unlikely
		if strings.HasPrefix(writeErr.Error(), "pq: duplicate key value violates unique constraint") {
			log.Println("Duplicate ID generated, trying again....")
			processPodcast(feed)
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
