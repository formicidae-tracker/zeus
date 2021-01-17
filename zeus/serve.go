package main

import (
	"fmt"

	flags "github.com/jessevdk/go-flags"
)

type ServeCommand struct {
	Args struct {
		Config flags.Filename
	} `positional-args:"yes"`
}

func (c *ServeCommand) Execute(args []string) error {
	return fmt.Errorf("Not yet implemented")
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
