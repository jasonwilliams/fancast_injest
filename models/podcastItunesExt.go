package models

// PodcastItunesExt represents the itunes metadata
type PodcastItunesExt struct {
	// Duration - represents audio length
	Duration string `db:"duration" json:"duration" `
}
