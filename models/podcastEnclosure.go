package models

// PodcastEnclosure represents the actual audio file and destination
type PodcastEnclosure struct {
	// URL - represents where the Audio artifact lives
	URL string `db:"url" json:"url" `
	// Type - What mime type the audio is e.g "audio/mpeg"
	Type string `db:"type" json:"type" `
	// Length - How long the audio is in seconds e.g 11584000
	Length string `db:"length" json:"length"`
}
