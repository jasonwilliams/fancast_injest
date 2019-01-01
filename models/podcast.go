package models

import (
	"encoding/json"
	"time"

	"bitbucket.org/jayflux/mypodcasts_injest/logger"
)

// Podcast represents the structure of a podcast
type Podcast struct {
	ID          string           `db:"id" json:"id"`
	Title       string           `db:"title" json:"title"`
	Description string           `db:"description" json:"description"`
	Image       json.RawMessage  `db:"image" json:"image"`
	Episodes    []PodcastEpisode `db:"episodes" json:"episodes"`
}

// GetPodcast returns a Podcast struct
func GetPodcast(id string) Podcast {
	var podcast Podcast
	row := db.QueryRow("SELECT id, title, description, image FROM podcasts where id = $1", id)
	err := row.Scan(&podcast.ID, &podcast.Title, &podcast.Description, &podcast.Image)
	if err != nil {
		logger.Log.Println(err)
	}

	podcast.Episodes = podcast.GetEpisodes()
	return podcast
}

// GetUpdatedPodcasts returns a list of podcasts ordered by last changed
func GetUpdatedPodcasts() []Podcast {
	var podcasts []Podcast
	// Select all podcast episodes ordered by published then return the brand
	rows, err := db.Query("select podcasts.id, podcasts.title, podcasts.description, podcasts.image from podcast_episodes inner join podcasts ON (podcast_episodes.parent = podcasts.id) order by published_parsed desc LIMIT 20")
	if err != nil {
		logger.Log.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var podcast Podcast
		if err := rows.Scan(&podcast.ID, &podcast.Title, &podcast.Description, &podcast.Image); err != nil {
			logger.Log.Fatal(err)
		}

		podcasts = append(podcasts, podcast)
	}

	return podcasts
}

// GetEpisodes fetches the first 20 episodes related to this podcast
func (p Podcast) GetEpisodes() []PodcastEpisode {
	// First lets get a date from the past
	datetime, err := time.Parse(time.RFC3339, "1990-08-24T11:00:00Z")
	if err != nil {
		logger.Log.Println(err)
	}

	podcastEpisodes := GetPodcastEpisodes(p.ID, datetime)
	return podcastEpisodes

}
