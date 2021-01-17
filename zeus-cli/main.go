package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

type Options struct {
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func Execute() error {
	if _, err := parser.Parse(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := Execute(); err != nil {
		if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
			return
		}
		os.Exit(1)
	}
}
