package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Interface       string        `long:"interface" short:"i" description:"Interface to use" default:"slcan0"`
	ID              uint8         `long:"id" description:"ID to calibrate" default:"1"`
	Temperature     float64       `long:"temperature" short:"t" description:"calibration temperature" default:"26.0"`
	Duration        time.Duration `long:"duration" short:"d" description:"time to wait to reach desired temperature" default:"2h"`
	ReferenceSensor uint8         `long:"reference-sensor" short:"r" description:"Select a sensor as reference, if 0 mean of tmp1075 is used" default:"0"`
}

func Execute() error {
	opts := Options{}

	if _, err := flags.Parse(&opts); err != nil {
		return err
	}

	intf, err := socketcan.NewRawInterface(opts.Interface)
	if err != nil {
		return err
	}

	frames := make(chan arke.ReceivableMessage)
	go func() {
		defer func() {
			close(frames)
		}()

		messageWhiteList := map[arke.MessageClass]struct{}{
			arke.ZeusReportMessage: struct{}{},
			arke.HeartBeatMessage:  struct{}{},
		}

		for {
			f, err := intf.Receive()
			if err != nil {
				log.Printf("CAN Receive error: %s", err)
			}
			m, id, err := arke.ParseMessage(&f)
			if err != nil {
				log.Printf("Arke Parsing error: %s", err)
			}
			if id != arke.NodeID(opts.ID) {
				continue
			}
			if _, ok := messageWhiteList[m.MessageClassID()]; ok == false {
				continue
			}

			frames <- m
		}

	}()
	heartbeats := make(chan *arke.HeartBeatData)

	go func() {
		for {
			tick := time.NewTicker(5 * time.Second)
			select {
			case f, ok := <-frames:
				if ok == false {
					return
				}
				switch f.MessageClassID() {
				case arke.HeartBeatMessage:
					heartbeats <- f.(*arke.HeartBeatData)
				}
			case <-tick.C:
				panic(fmt.Sprintf("Connection to Zeus %d timeouted", opts.ID))
			}
		}
	}()

	if err := arke.Ping(intf, arke.ZeusClass); err != nil {
		return err
	}

	<-heartbeats

	log.Printf("Found Zeus Node %d", opts.ID)

	return nil
}

func main() {

	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
