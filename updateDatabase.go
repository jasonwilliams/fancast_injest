package main

import (
	"fmt"
	"github.com/minio/minio-go"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

var accessKey string = os.Getenv("SPACES_KEY")
var secKey string = os.Getenv("SPACES_SECRET_KEY")
var ssl bool = true

func updateDatabase() {
	log.Println("[Restore] - Starting restore")
	now := time.Now()
	var difference float64 = math.MaxFloat64
	var latestObject minio.ObjectInfo

	// Initiate a client using DigitalOcean Spaces.
	client, err := minio.New("ams3.digitaloceanspaces.com", accessKey, secKey, ssl)
	if err != nil {
		log.Fatal(err)
	}

	// https://docs.minio.io/docs/golang-client-api-reference#ListObjects
	// Loop through objests and get latest one
	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	isRecursive := true
	// There isn't an easy way to get the latest object, so we need to loop through to find the latest one
	objectCh := client.ListObjectsV2("fancast", "database-backups/", isRecursive, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}

		if now.Sub(object.LastModified).Seconds() < difference {
			difference = now.Sub(object.LastModified).Seconds()
			latestObject = object
		}
	}

	log.Println("Downloading latest database backup...")
	err = client.FGetObject("fancast", latestObject.Key, "./fancast.backup", minio.GetObjectOptions{})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("downloaded %s, updating database", latestObject.Key)
	cmd := exec.Command("sudo", "-u", "fancast", "pg_restore", "-c", "-d", "fancast", "--if-exists", "./fancast.backup")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	log.Printf("%s", err)
}

func performBackup() {
	// We should add a time to the DB backup
	// TODO Not local time
	// https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format
	t := time.Now().Format("2006-01-02T15-04-05")
	fileName := "fancast-" + t + ".backup"

	// Perform DB Backup
	cmd := exec.Command("sudo", "-u", "fancast", "pg_dump", "-f", fileName, "-Fc", "fancast")
	cmd.Run()

	log.Println("[backup] Writing of " + fileName + "  complete")

	// Initiate a client using DigitalOcean Spaces.
	client, err := minio.New("ams3.digitaloceanspaces.com", accessKey, secKey, ssl)
	if err != nil {
		log.Fatal(err)
	}

	// https://docs.minio.io/docs/golang-client-api-reference#FPutObject
	n, err := client.FPutObject("fancast", "database-backups/"+fileName, "./"+fileName, minio.PutObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Println("[backup] Upload of " + fileName + " complete")
	log.Println("Successfully uploaded bytes: ", n)
	// Delete file once uploaded
	// os.Remove(fileName)

}
