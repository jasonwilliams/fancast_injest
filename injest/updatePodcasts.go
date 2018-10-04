package injest

// UpdateNewPodcasts updates new podcasts
func UpdateNewPodcasts() {
	// Fetch podcasts which haven't had their last_change set (new podcasts)
	// This should be a one-off
	var feedURL string
	// Create channel to put our URLS into
	// We don't want to overload the injestor, so lets buffer to 5
	urls := make(chan string, 5)
	status := make(chan int)
	go Injest(urls, status)

	rows, err := db.Query("select feed_url from podcasts where last_change is NULL")
	if err != nil {
		log.Println(err)
		log.Fatal("UpdatePodcasts: error in query")
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&feedURL)
		urls <- feedURL
	}
}

// UpdatePodcasts updates podcasts which need updating
func UpdatePodcasts() {
	// Fetch podcasts which haven't had their last_change set (new podcasts)
	// This should be a one-off
	var feedURL string
	// Create channel to put our URLS into
	// We don't want to overload the injestor, so lets buffer to 5
	urls := make(chan string, 5)
	status := make(chan int)
	go Injest(urls, status)

	rows, err := db.Query("select feed_url from podcasts where extract('epoch' from age(now(), last_fetch))/3600 > poll_frequency;")
	if err != nil {
		log.Println(err)
		log.Fatal("UpdatePodcasts: error in query")
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&feedURL)
		urls <- feedURL
	}
}
