package main

import (
	"os"

	flags "github.com/jessevdk/go-flags"
)

type Options struct {
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func Execute() error {
	_, err := parser.Parse()
	if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
		return nil
	}
	return err
}

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
