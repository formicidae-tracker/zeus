package main

import (
	"log"
	"net/http"
	"os"
)

func Execute() error {

	http.Handle("/", http.FileServer(http.Dir("./webapp/dist/webapp")))
	return http.ListenAndServe(":3000", nil)
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
