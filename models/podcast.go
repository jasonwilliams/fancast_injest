package models

import (
	"encoding/json"
	"fmt"
)

// Podcast represents the structure of a podcast
type Podcast struct {
	ID          string          `db:"id" json:"id"`
	Title       string          `db:"title" json:"title"`
	Description string          `db:"description" json:"description"`
	Image       json.RawMessage `db:"image" json:"image"`
}

// GetPodcast returns a Podcast struct
func GetPodcast(id string) Podcast {
	var podcast Podcast
	row := db.QueryRow("SELECT id, title, description, image FROM podcasts where id = $1", id)
	err := row.Scan(&podcast.ID, &podcast.Title, &podcast.Description, &podcast.Image)
	if err != nil {
		fmt.Println(err)
	}
	return podcast
}

// GetImage gets the podcast image for this current Podcast struct
func (p Podcast) GetImage() (PodcastImage, error) {
	var podcastImage PodcastImage
	error := json.Unmarshal(p.Image, podcastImage)
	return podcastImage, error
}
