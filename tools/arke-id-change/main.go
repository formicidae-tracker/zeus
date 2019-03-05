package main

import (
	"log"
	"os"
)

type Options struct {
	Interface string `long:"interface" short:"i" description:"interface to use" default:"slcan0"`
	Class     string `long:"class" short:"c" description:"class of device to change"`
	Original  uint8  `long:"original" short:"o" description:"orginal ID to change" default:"0"`
	Original  target `long:"original" short:"o" description:"orginal ID to change" default:"0"`
}

func Execute() error {
	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
