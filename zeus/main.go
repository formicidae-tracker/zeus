package main

import (
	"fmt"
	"os"

	"github.com/formicidae-tracker/zeus"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Version bool `short:"V" long:"version" description:"Prints current version"`
	Args    struct {
		Config flags.Filename
	} `positional-args:"yes"`
}

func Execute() error {
	opts := &Options{}
	if _, err := flags.Parse(opts); err != nil {
		return err
	}

	if opts.Version == true {
		fmt.Println(zeus.ZEUS_VERSION)
		os.Exit(0)
	}

	return fmt.Errorf("Not yet implemented")
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[zeus] Unhandled error: %s\n", err)
		os.Exit(1)
	}
}
