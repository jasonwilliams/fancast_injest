package models

import (
	"encoding/json"
	"time"

	"bitbucket.org/jayflux/mypodcasts_injest/logger"
)

// PodcastEpisode represents the structure of a podcast
type PodcastEpisode struct {
	ID              string          `db:"id" json:"id"`
	Title           string          `db:"title" json:"title"`
	Description     string          `db:"description" json:"description"`
	Image           json.RawMessage `db:"image" json:"image"`
	PublishedParsed string          `db:"published_parsed" json:"publishedParsed"`
}

// GetPodcastEpisode returns a Podcast struct
func GetPodcastEpisode(id string) PodcastEpisode {
	var podcastEpisode PodcastEpisode
	row := db.QueryRow("SELECT id, title, description, image, published_parsed FROM podcast_episodes where id = $1", id)
	row.Scan(&podcastEpisode.ID, &podcastEpisode.Title, &podcastEpisode.Description, &podcastEpisode.Image, &podcastEpisode.PublishedParsed)
	return podcastEpisode
}

// GetPodcastEpisodes returns multiple episodes based on a datetime
// Example datetime from database - 2018-08-24T11:00:00Z
func GetPodcastEpisodes(id string, datetime time.Time) []PodcastEpisode {
	var podcastEpisodes []PodcastEpisode
	rows, err := db.Query("SELECT id, title, description, image, published_parsed FROM podcast_episodes where parent = $1 AND published_parsed > $2 LIMIT 20", id, datetime)
	if err != nil {
		logger.Log.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var podcastEpisode PodcastEpisode
		if err := rows.Scan(&podcastEpisode.ID, &podcastEpisode.Title, &podcastEpisode.Description, &podcastEpisode.Image, &podcastEpisode.PublishedParsed); err != nil {
			logger.Log.Fatal(err)
		}
		podcastEpisodes = append(podcastEpisodes, podcastEpisode)
	}

	return podcastEpisodes
}

// GetImage gets the podcast image for this current Podcast struct
func (p PodcastEpisode) GetImage() (PodcastImage, error) {
	var podcastImage PodcastImage
	error := json.Unmarshal(p.Image, podcastImage)
	return podcastImage, error
}
