package main

import (
	"flag"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
	"bitbucket.org/jayflux/mypodcasts_injest/injestFromDataset"
)

var build = flag.String("build", "", "Specify type of build")

func main() {
	// Parse commandline arguments
	flag.Parse()
	switch *build {
	case "injest":
		injest.Injest(flag.Arg(0))
	case "tsv":
		injestFromDataset.CrawlDataset()
	}

}
