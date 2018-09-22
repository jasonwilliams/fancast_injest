package models

import (
	"database/sql"
	"fmt"
	"log"

	"bitbucket.org/jayflux/mypodcasts_injest/logger"
	// Needed for database/sql
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var (
	db      *sql.DB
	connErr error
)

// Initialises the database, was originlly init() but this ran too fast
func InitDB() {
	// Connect to the database
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s", viper.Get("database.user"), viper.Get("database.database"), viper.Get("database.password"))
	db, connErr = sql.Open("postgres", connStr)
	if connErr != nil {
		log.Fatal(connErr)
	}
	logger.Log.Println("Connected to database")
}
