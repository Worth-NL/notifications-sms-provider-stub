package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", HealthcheckEndpoint)
	http.HandleFunc("/mmg", MmgEndpoint)
	http.HandleFunc("/firetext", FiretextEndpoint)
	http.HandleFunc("/spryng", SpryngEndpoint)
	port := getenv("PORT", "6300")
	log.Printf("Listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
