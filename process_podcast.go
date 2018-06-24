package main

import (
	"flag"
	"fmt"
	"time"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
	"bitbucket.org/jayflux/mypodcasts_injest/injestFromDataset"
	"github.com/spf13/viper"
	"gopkg.in/robfig/cron.v2"
)

var build = flag.String("build", "", "Specify type of build")
var DB = flag.String("db", "", "update or backup")
var updater = flag.Bool("cron", false, "Initiate application")

func main() {
	// Setup Config
	setupConfig()

	// Parse commandline arguments
	flag.Parse()
	switch *build {
	case "injest":
		urls := make(chan string, 1)
		status := make(chan int)
		urls <- flag.Arg(0)
		go injest.Injest(urls, status)
		close(urls)
		<-status
	case "tsv":
		injestFromDataset.CrawlDataset()
	}

	switch *DB {
	case "update":
		updateDatabase()
	case "backup":
		performBackup()
	}

	// Set up cron job to do various tasks, including backing up database
	if *updater {
		// https://godoc.org/gopkg.in/robfig/cron.v2
		fmt.Println("Hello")
		c := cron.New()
		c.AddFunc("@hourly", func() { performBackup() })
		go forever()
		select {}
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

func forever() {
	for {
		time.Sleep(time.Second)
	}
}
