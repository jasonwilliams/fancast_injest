package models

// PodcastImage represents the structure of a podcast
type PodcastImage struct {
	// The ID of this image is the has which maps to the bucket
	ID string `db:"id" `
	// Title, most often used for alt,title tags
	Title string `db:"title"`
	// URL of the original image taken from the RSS feed
	URL string `db:"url"`
	// Image extention, images could be png,jpg
	Ext string `db:"ext"`
}
