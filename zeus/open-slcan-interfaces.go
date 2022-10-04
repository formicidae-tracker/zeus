package main

import (
	"fmt"
	"os"
	"os/signal"

	flags "github.com/jessevdk/go-flags"
)

type OpenSlcanInterfacesCommand struct {
	Args struct {
		Config flags.Filename
	} `positional-args:"yes"`
}

func (c *OpenSlcanInterfacesCommand) Execute(args []string) error {
	config, err := OpenConfigFromArg(c.Args.Config)
	if err != nil {
		return err
	}
	if err = config.Check(); err != nil {
		return err
	}
	managers := map[string]*SlcandManager{}
	defer func() {
		for ifname, manager := range managers {
			err = manager.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[zeus] could not close %s:%s\n", ifname, err)
			}
		}
	}()
	for ifname, devname := range config.Interfaces {
		manager, err := OpenSlcand(ifname, devname)
		if err != nil {
			return err
		}
		managers[ifname] = manager
	}
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	return nil
}

func init() {
	_, err := parser.AddCommand("open-slcan-interfaces",
		"open slcan interfaces",
		"Opens slcan interfaces from this computer. It requires super user right",
		&OpenSlcanInterfacesCommand{})
	if err != nil {
		panic(err.Error())
	}
}
