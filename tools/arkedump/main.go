package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/jessevdk/go-flags"
)

type Options struct {
}

func Execute() error {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.Usage = "[OPTIONS] interface_name"

	args, err := parser.Parse()
	if err != nil {
		return err
	}
	if len(args) != 1 {
		parser.WriteHelp(os.Stderr)
		return fmt.Errorf("Need a signle interface name as argument")
	}

	intf, err := socketcan.NewRawInterface(args[0])
	if err != nil {
		return err
	}

	frames := make(chan socketcan.CanFrame, 1)

	go func() {
		defer close(frames)
		for {
			f, err := intf.Receive()
			if err != nil {
				if errno, ok := err.(syscall.Errno); ok == true {
					if errno == syscall.EBADF || errno == syscall.ENETDOWN || errno == syscall.ENODEV {
						log.Printf("Closed CAN Interface: %s", err)
						return
					}
				}
				log.Printf("Could not receive CAN frame on: %s", err)
				frames <- f
			}
		}
	}()

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		intf.Close()
	}()

	for f := range frames {
		m, ID, err := arke.ParseMessage(&f)
		if err != nil {
			fmt.Fprintf(os.Stdout, "%s ID:%d %+v", time.Now(), ID, m)
		} else {
			log.Printf("Could not parse CAN Frame: %s", err)
		}
	}

	return nil
}

func main() {
	if err := Execute(); err != nil {
		if flags.WroteHelp(err) == true {
			os.Exit(0)
		}
		log.Fatalf("Unhandled error: %s", err)
	}
}
