package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"bitbucket.org/jayflux/mypodcasts_injest/models"

	"github.com/gorilla/mux"
	// Needed for database/sql
	_ "github.com/lib/pq"
)

// API Entrypoint to the API
func API() {
	router := mux.NewRouter()
	router.HandleFunc("/test", Test).Methods("GET")
	router.HandleFunc("/podcasts/{podcast}", podcastHandler)
	router.HandleFunc("/podcasts/e/{podcast}", podcastEpisodeHandler)
	router.HandleFunc("/podcasts/{podcast}/episodes", podcastEpisodesHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8060", router))
}

// Test is a testing function
func Test(w http.ResponseWriter, r *http.Request) {

}

// Handle the podcast homepage
func podcastHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podcast := models.GetPodcast(vars["podcast"])
	podcastJSON, _ := json.Marshal(podcast)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(podcastJSON))

}

// Handle the podcast homepage
func podcastEpisodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podcast := models.GetPodcastEpisode(vars["podcast"])
	podcastJSON, _ := json.Marshal(podcast)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(podcastJSON))

}

// Handle fetching episodes for a podcast
func podcastEpisodesHandler(w http.ResponseWriter, r *http.Request) {

}
