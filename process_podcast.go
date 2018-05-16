package main

import (
	"flag"
	"fmt"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
	"bitbucket.org/jayflux/mypodcasts_injest/injestFromDataset"
	"github.com/spf13/viper"
)

var build = flag.String("build", "", "Specify type of build")

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
