package models

import (
	"encoding/json"
)

// PodcastEpisode represents the structure of a podcast
type PodcastEpisode struct {
	ID          string          `db:"id" json:"id"`
	Title       string          `db:"title" json:"title"`
	Description string          `db:"description" json:"description"`
	Image       json.RawMessage `db:"image" json:"image"`
}

// GetPodcastEpisode returns a Podcast struct
func GetPodcastEpisode(id string) PodcastEpisode {
	var podcastEpisode PodcastEpisode
	row := db.QueryRow("SELECT id, title, description, image FROM podcast_episodes where id = $1", id)
	row.Scan(&podcastEpisode.ID, &podcastEpisode.Title, &podcastEpisode.Description, &podcastEpisode.Image)
	return podcastEpisode
}

// GetImage gets the podcast image for this current Podcast struct
func (p PodcastEpisode) GetImage() (PodcastImage, error) {
	var podcastImage PodcastImage
	error := json.Unmarshal(p.Image, podcastImage)
	return podcastImage, error
}
