package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// API Entrypoint to the API
func API() {
	router := mux.NewRouter()
	router.HandleFunc("/test", Test).Methods("GET")
	log.Fatal(http.ListenAndServe("0.0.0.0:8060", router))
}

// Test is a testing function
func Test(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "", "hello world")
}
