package injestFromDataset

import (
	"encoding/csv"
	"log"
	"os"

	"bitbucket.org/jayflux/mypodcasts_injest/injest"
)

func CrawlDataset() {
	file, _ := os.Open("/var/local/all-podcasts-dataset/a.tsv")
	defer file.Close()

	r := csv.NewReader(file)
	r.Comma = '\t'

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Create channel to put our URLS into
	// We don't want to overload the injestor, so lets buffer to 5
	urls := make(chan string)
	status := make(chan int)
	go injest.Injest(urls, status)

	for _, each := range records {
		urls <- each[3]
	}
	close(urls)

}
