package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// API Entrypoint to the API
func API() {
	router := mux.NewRouter()
	router.HandleFunc("/test", Test).Methods("GET")
	router.HandleFunc("/podcasts/{podcast}", podcastHandler)
	router.HandleFunc("/podcasts/{podcast}/episodes", podcastEpisodesHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8060", router))
}

// Test is a testing function
func Test(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "", "hello world")
}

// Handle the podcast homepage
func podcastHandler(w http.ResponseWriter, r *http.Request) {

}

// Handle fetching episodes for a podcast
func podcastEpisodesHandler(w http.ResponseWriter, r *http.Request) {

}
