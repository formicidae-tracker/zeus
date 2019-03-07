package main

import (
	"log"
	"os"
)

func Execute() error {
	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
