package main

import (
	"bufio"
	"fmt"
	"github.com/minio/minio-go"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

var accessKey string = os.Getenv("SPACES_KEY")
var secKey string = os.Getenv("SPACES_SECRET_KEY")
var ssl bool = true

func updateDatabase() {
	// Initiate a client using DigitalOcean Spaces.
	client, err := minio.New("ams3.digitaloceanspaces.com", accessKey, secKey, ssl)
	if err != nil {
		log.Fatal(err)
	}

	// https://docs.minio.io/docs/golang-client-api-reference#ListObjects
	err = client.FGetObject("fancast", "database-backups/mypodcasts.gz", "./mypodcasts.gz", minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
}

func performBackup() {
	// We should add a time to the DB backup
	// TODO Not local time
	// https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format
	t := time.Now().Format("2006-01-02T15-04-05")
	fileName := "fancast-" + t + ".gz"

	// First create file to put DB Output into
	f, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	// Perform DB Backup
	cmd := exec.Command("sudo", "-u", "developer", "pg_dump", "fancast")
	gzip := exec.Command("gzip")

	r, w := io.Pipe()
	cmd.Stdout = w
	gzip.Stdin = r

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	gzip.Stdout = writer

	// https://stackoverflow.com/questions/10781516/how-to-pipe-several-commands-in-go
	cmd.Start()
	gzip.Start()
	cmd.Wait()
	w.Close()
	gzip.Wait()

	log.Println("[backup] Writing of " + fileName + "  complete")

	// Initiate a client using DigitalOcean Spaces.
	client, err := minio.New("ams3.digitaloceanspaces.com", accessKey, secKey, ssl)
	if err != nil {
		log.Fatal(err)
	}

	// https://docs.minio.io/docs/golang-client-api-reference#FPutObject
	_, err = client.FPutObject("fancast", "database-backups/"+fileName, "./"+fileName, minio.PutObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Println("[backup] Upload of " + fileName + " complete")

}
