package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"git.tuleu.science/fort/libarke/src-go/arke"

	"github.com/atuleu/golang-socketcan"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Interface string `long:"interface" short:"i" description:"interface to use" default:"slcan0"`
	Class     string `long:"class" short:"c" description:"class of device to change"`
	Original  uint8  `long:"original" short:"o" description:"orginal ID to change" default:"0"`
	Target    uint8  `long:"target" short:"t" description:"target ID" default:"0"`
}

func Execute() error {
	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	class := map[string]arke.NodeClass{
		"zeus":    arke.ZeusClass,
		"celaeno": arke.CelaenoClass,
		"helios":  arke.HeliosClass,
	}

	c, ok := class[strings.ToLower(opts.Class)]
	if ok == false {
		return fmt.Errorf("Unknown class '%s'", opts.Class)
	}

	intf, err := socketcan.NewRawInterface(opts.Interface)
	if err != nil {
		return err
	}

	return arke.SendIDChangeRequest(intf, c, arke.NodeID(opts.Original), arke.NodeID(opts.Target))
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
