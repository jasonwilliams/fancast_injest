package injest

// UpdateNewPodcasts updates new podcasts
func UpdateNewPodcasts() {
	// Fetch podcasts which haven't had their last_change set (new podcasts)
	// This should be a one-off
	var feedURL string

	rows, err := db.Query("select feed_url from podcasts where last_change is NULL")
	if err != nil {
		log.Println(err)
		log.Fatal("UpdatePodcasts: error in query")
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&feedURL)
		Injest(feedURL)
	}
}

// UpdatePodcasts updates podcasts which need updating
func UpdatePodcasts() {
	log.Println("Performing update on podcasts..")
	// Fetch podcasts which haven't had their last_change set (new podcasts)
	// This should be a one-off
	var feedURL string

	rows, err := db.Query("select feed_url from podcasts where extract('epoch' from age(now(), last_fetch))/3600 > poll_frequency")
	if err != nil {
		log.Println(err)
		log.Fatal("UpdatePodcasts: error in query")
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&feedURL)
		Injest(feedURL)
	}
}

// UpdatePollFrequencies will go through all podcasts and set the right polling frequency
func UpdatePollFrequencies() {
	var feedURL string
	// Fetch all podcasts and update their poll frequencies
	rows, err := db.Query("select feed_url from podcasts")
	if err != nil {
		log.Println(err)
		log.Fatal("UpdatePollFrequencies: error in query")
	}
	defer rows.Close()
	tx, err := db.Begin()
	if err != nil {
		log.Println("UpdatePollFrequencies: Couldn't begin database transaction")
		log.Fatal(err)
	}

	for rows.Next() {
		rows.Scan(&feedURL)
		freq := updatePollFrequency(feedURL)
		_, writeErr := tx.Exec("UPDATE podcasts SET poll_frequency = $1 WHERE feed_url = $2", freq, feedURL)
		if writeErr != nil {
			log.Println("UpdatePollFrequencies: Could not write to DB")
			log.Fatal(writeErr)
		}
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("UpdatePollFrequencies: Commit failed")
		log.Fatal(commitErr)
	}

}
