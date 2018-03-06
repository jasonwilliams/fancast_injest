package injest

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"

	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

// For performance, compile this once at the beginning
var UUIDRegex = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func Injest(url string) {
	// Setup Config
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   //
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// Connect to the database
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s", viper.Get("database.user"), viper.Get("database.database"), viper.Get("database.password"))
	db, connErr = sql.Open("postgres", connStr)
	if connErr != nil {
		log.Fatal(connErr)
	}

	// make sure we're fetching the correct URL, if there's been a 301, this will use the new endpoint
	// This function will also update the DB if there has been a redirect
	url = checkPodcastUrl(url)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Printf("Injest: Error parsing %s\n", url)
		log.Fatal(err)
	}

	process(feed, url)
}
