package models

import (
	"encoding/json"
	"time"

	"bitbucket.org/jayflux/mypodcasts_injest/logger"
)

// The format of Published in the database - using reference time
var episodePublishedFormat = "Mon, 2 Jan 2006 15:04:05 -0700"

// The format of the API output - using reference time
var episodePublishedOutputFormat = "Jan 02, 2006"

// PodcastEpisode represents the structure of a podcast
type PodcastEpisode struct {
	ID              string          `db:"id" json:"id"`
	Title           string          `db:"title" json:"title"`
	Description     string          `db:"description" json:"description"`
	Image           json.RawMessage `db:"image" json:"image"`
	PublishedParsed string          `db:"published_parsed" json:"publishedParsed"`
	Published       string          `db:"published" json:"published"`
	ParentID        string          `db:"parentID" json:"parentID"`
	ParentTitle     string          `db:"parentTitle" json:"parentTitle"`
	Enclosures      json.RawMessage `db:"enclosures" json:"enclosures"`
	ItunesExt       json.RawMessage `db:"itunes_ext" json:"itunes_ext"`
	Length          string          `json:"length"`
}

// GetPodcastEpisode returns a Podcast struct
func GetPodcastEpisode(id string) PodcastEpisode {
	var podcastEpisode PodcastEpisode
	row := db.QueryRow("SELECT podcast_episodes.id, podcast_episodes.title, podcast_episodes.description, COALESCE(NULLIF(podcast_episodes.image, 'null'::jsonb), podcasts.image) AS image, podcast_episodes.published_parsed, podcast_episodes.published, podcast_episodes.parent, podcast_episodes.enclosures, podcasts.title AS parentTitle FROM podcast_episodes INNER JOIN podcasts ON (podcast_episodes.parent = podcasts.id) where podcast_episodes.id = $1", id)
	row.Scan(&podcastEpisode.ID, &podcastEpisode.Title, &podcastEpisode.Description, &podcastEpisode.Image, &podcastEpisode.PublishedParsed, &podcastEpisode.Published, &podcastEpisode.ParentID, &podcastEpisode.Enclosures, &podcastEpisode.ParentTitle)

	// Set the proper formatting for published
	podcastEpisode.formatPublished()
	return podcastEpisode
}

// GetPodcastEpisodes returns multiple episodes based on a datetime
// Example datetime from database - 2018-08-24T11:00:00Z
func GetPodcastEpisodes(id string, datetime time.Time) []PodcastEpisode {
	var podcastEpisodes []PodcastEpisode
	rows, err := db.Query("SELECT podcast_episodes.id, podcast_episodes.title, podcast_episodes.description, COALESCE(NULLIF(podcast_episodes.image, 'null'::jsonb), podcasts.image) AS image, podcast_episodes.published_parsed, podcast_episodes.published, podcast_episodes.enclosures, podcast_episodes.itunes_ext FROM podcast_episodes INNER JOIN podcasts ON (podcast_episodes.parent = podcasts.id) where podcast_episodes.parent = $1 AND published_parsed > $2 ORDER BY published_parsed DESC LIMIT 20", id, datetime)
	if err != nil {
		logger.Log.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var podcastEpisode PodcastEpisode
		if err := rows.Scan(&podcastEpisode.ID, &podcastEpisode.Title, &podcastEpisode.Description, &podcastEpisode.Image, &podcastEpisode.PublishedParsed, &podcastEpisode.Published, &podcastEpisode.Enclosures, &podcastEpisode.ItunesExt); err != nil {
			logger.Log.Fatal(err)
		}
		// Set the proper formatting for published
		podcastEpisode.formatPublished()
		podcastEpisode.getLength()
		podcastEpisodes = append(podcastEpisodes, podcastEpisode)
	}

	return podcastEpisodes
}

func (p *PodcastEpisode) formatPublished() {
	timeStr, err := time.Parse(episodePublishedFormat, p.Published)
	if err != nil {
		logger.Log.Println(err)
	}
	p.Published = timeStr.Format(episodePublishedOutputFormat)
}

func (p *PodcastEpisode) getLength() {
	var itunes PodcastItunesExt
	err := json.Unmarshal(p.ItunesExt, &itunes)
	if err != nil {
		logger.Log.Println(err)
	}

	p.Length = itunes.Duration
}
