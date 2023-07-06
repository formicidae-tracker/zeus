package main

import (
	"fmt"

	"github.com/formicidae-tracker/zeus/internal/zeus"
	flags "github.com/jessevdk/go-flags"
)

type VersionCommand struct {
	Args struct {
		Config flags.Filename
	} `positional-args:"yes"`
}

func (c *VersionCommand) Execute(args []string) error {
	fmt.Printf("%s\n", zeus.ZEUS_VERSION)
	return nil
}

func init() {
	_, err := parser.AddCommand("version",
		"print zeus version",
		"prints zeus version on stdout and exit",
		&VersionCommand{})
	if err != nil {
		panic(err.Error())
	}
}
