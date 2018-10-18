package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/pprof"

	"bitbucket.org/jayflux/mypodcasts_injest/api"
	"bitbucket.org/jayflux/mypodcasts_injest/models"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
	"bitbucket.org/jayflux/mypodcasts_injest/injestFromBBC"
	"bitbucket.org/jayflux/mypodcasts_injest/logger"
	"github.com/spf13/viper"
	"gopkg.in/robfig/cron.v2"
)

var build = flag.String("build", "", "Specify type of build")
var dbFlag = flag.String("db", "", "update or backup")
var updater = flag.Bool("cron", false, "Initiate application")
var apiFlag = flag.Bool("api", false, "Start API")
var cpuprofile = flag.Bool("cpuprofile", false, "write cpu profile to file")
var log = logger.Log

func main() {
	// Setup Config
	setupConfig()

	// Setup logging
	f, err := os.OpenFile("/var/log/fancast/error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.Println("Application started")

	// Parse commandline arguments
	flag.Parse()

	// Setup CPU Profiling
	if *cpuprofile {
		log.Println("profiling...")
		f, err := os.Create("./testing.prof")
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	switch *build {
	case "injest":
		urls := make(chan string, 1)
		status := make(chan int)
		urls <- flag.Arg(0)
		go injest.Injest(urls, status)
		close(urls)
		<-status
	case "bbc":
		injestFromBBC.CrawlBBC()

	case "update":
		injest.UpdatePodcasts()
	}

	switch *dbFlag {
	case "update":
		updateDatabase()
	case "backup":
		performBackup()
	}

	// Set up cron job to do various tasks, including backing up database
	if *updater {
		// https://godoc.org/gopkg.in/robfig/cron.v2
		c := cron.New()
		c.AddFunc("@hourly", func() { injest.UpdatePodcasts() })
		c.AddFunc("@hourly", func() {
			cmd := exec.Command("node", "imageProcessing/updateImages.js")
			err := cmd.Run()
			if err != nil {
				log.Println(err)
			}
		})
		c.AddFunc("@weekly", func() { injestFromBBC.CrawlBBC() })
		c.Start()
		select {}
	}

	if *apiFlag {
		models.InitDB()
		api.API()
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
