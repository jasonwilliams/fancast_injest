package logger

import (
	"log"
	"os"
)

// Log is the main logger of our application
var Log *log.Logger

// Init starts when the first package imports it
func init() {
	// Location of log file
	logpath := "./combined.log"
	f, err := os.OpenFile(logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	Log = log.New(f, "", log.LstdFlags|log.Lshortfile)
	Log.Println("LogFile : " + logpath)
	if os.Getenv("NODE_ENV") == "development" {
		Log.SetOutput(os.Stderr)
		Log.SetOutput(os.Stdout)
	}
}
