package models

// PodcastEnclosure represents the actual audio file and destination
type PodcastEnclosure struct {
	// URL - represents where the Audio artifact lives
	URL string `db:"url" `
	// Type - What mime type the audio is e.g "audio/mpeg"
	Type string `db:"type"`
	// Length - How long the audio is in seconds e.g 11584000
	Length int64 `db:"length"`
}
