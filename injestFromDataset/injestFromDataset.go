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

	for _, each := range records {
		injest.Injest(each[3])
	}

}
