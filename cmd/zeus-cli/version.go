package main

import (
	"fmt"
	"os"

	"github.com/formicidae-tracker/zeus/internal/zeus"
)

type VersionCommand struct {
}

func (c *VersionCommand) Execute(args []string) error {
	fmt.Fprintf(os.Stdout, "zeus-cli version %s\n", zeus.ZEUS_VERSION)
	return nil
}

func init() {
	_, err := parser.AddCommand("version",
		"print version",
		"prints version on stdout",
		&VersionCommand{})
	if err != nil {
		panic(err.Error())
	}

}
