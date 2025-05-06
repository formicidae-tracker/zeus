package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"

	socketcan "github.com/atuleu/golang-socketcan"
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

	rx := make(chan bool)
	go func() {
		for {
			f, err := intf.Receive()
			if err != nil {
				log.Printf("Could not receive CAN Frame: %s", err)
				continue
			}

			m, ID, err := arke.ParseMessage(&f)
			if err != nil {
				log.Printf("Could not receive Parse CAN frame: %s", err)
				continue
			}

			if m.MessageClassID() == arke.HeartBeatMessage && ID == arke.NodeID(opts.Original) && m.(*arke.HeartBeatData).Class == c {
				rx <- true
			}
		}
	}()

	if err := intf.Send(arke.MakePing(c)); err != nil {
		return err
	}

	timeout := time.NewTicker(2 * time.Second)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		return fmt.Errorf("Device '%s':'%d' seems unresponsive", c, opts.Original)
	case <-rx:
	}

	return intf.Send(arke.MakeIDChangeRequest(c, arke.NodeID(opts.Original), arke.NodeID(opts.Target)))
}

func main() {
	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
