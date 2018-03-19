package injestFromDataset

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
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
		fmt.Println(each[3])
	}

}
