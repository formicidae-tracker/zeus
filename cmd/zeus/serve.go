package main

import (
	"os"
	"os/signal"

	flags "github.com/jessevdk/go-flags"
)

type ServeCommand struct {
	Args struct {
		Config flags.Filename
	} `positional-args:"yes"`
}

func (c *ServeCommand) Execute(args []string) error {
	config, err := OpenConfigFromArg(c.Args.Config)
	if err != nil {
		return err
	}
	z, err := OpenZeus(*config)
	if err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	go z.run()
	<-sigint

	return z.shutdown()
}

func init() {
	_, err := parser.AddCommand("serve",
		"serve climate control",
		"serves climate control from this computer",
		&ServeCommand{})
	if err != nil {
		panic(err.Error())
	}
}
