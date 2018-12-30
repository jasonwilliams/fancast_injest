package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"bitbucket.org/jayflux/mypodcasts_injest/models"

	"github.com/gorilla/mux"
	// Needed for database/sql
	_ "github.com/lib/pq"
)

// API Entrypoint to the API
func API() {
	router := mux.NewRouter()
	router.HandleFunc("/test", Test).Methods("GET")
	// Get metadata about latest podcasts
	router.HandleFunc("/podcasts/latest", latestPodcastsHandler)
	// Get metadata about podcast
	router.HandleFunc("/podcasts/{podcast}", podcastHandler)
	// Get metadata about individual episode
	router.HandleFunc("/episodes/{podcast}", podcastEpisodeHandler)
	// Get multiple episodes from a podcast
	router.HandleFunc("/podcasts/{podcast}/episodes", podcastEpisodesHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8060", router))
}

// Test is a testing function
func Test(w http.ResponseWriter, r *http.Request) {

}

// Handle latest podcasts
func latestPodcastsHandler(w http.ResponseWriter, r *http.Request) {
	podcasts := models.GetUpdatedPodcasts()
	podcastsJSON, _ := json.Marshal(podcasts)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(podcastsJSON))
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
	vars := mux.Vars(r)
	q := r.URL.Query()
	dateTime, _ := time.Parse(time.RFC3339, q.Get("datetime"))
	podcast := models.GetPodcastEpisodes(vars["podcast"], dateTime)
	podcastJSON, _ := json.Marshal(podcast)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(podcastJSON))
}
