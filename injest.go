package main

import (
	// "database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
)

// For performance, compile this once at the beginning
const UUIDRegex = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func main() {
	// Setup Config
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   //
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("http://atp.fm/episodes?format=rss")
	if err != nil {
		log.Fatal(err)
	}

	processPodcast(feed)

}

// ProcessPodcast will take a feed object and start inserting the properties into the database
// It will also need to generate an ID for each podcast aswell
func processPodcast(feed *gofeed.Feed) {
	// Connect to the database
	connStr := fmt.Sprintf("user=%s dbname=%s", viper.Get("database.user"), viper.Get("database.database"))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	id = generateNewID()

	writeErr := db.Query("INSERT INTO podcast_episodes(id, title, description, published, publishedparsed, image, author, enclosures, duration, subtitle, summary) ")
	if writeErr != nil {
		log.println("Could not write to DB")
		log.Fatal(writeErr)
	}

	b, err := json.Marshal(feed.Image)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
}

// GUID's of podcasts can vary a LOT
// Some podcasts have perfectly good GUIDS, and we should be able to re-use these
// Some podcasts use the URL as their UUID, this isn't a great ID as it could change (https, change of TLD, change of mp3, new domain)
// Some podcasts don't have any GUID at all, in which case we need to generate one
func generateIDForPodcast(guid string) string {
	// This function could potentially be passed empty strings, its unlikely but can happen
	if len(s) != 0 {
		// We could be dealing with a Valid UUID, in which case we should just re-use it
		if IsValidUUID(guid) {
			return guid
		}

		id, err := uuid.FromString(guid)
		if err != nil {
			fmt.Printf("Something went wrong generating an ID: %s", err)
		}

		return id
	} else {
		return generateNewID()
	}
}

func generateNewID() string {
	id, err := uuid.NewV4()
	if err != nil {
		log.printf("Something went wrong generating a new ID: %s", err)
	}
	return id
}

// Check if a UUID is valid
func IsValidUUID(uuid string) bool {
	return UUIDRegex.MatchString(uuid)
}
