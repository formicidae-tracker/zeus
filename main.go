package main

import (
	"fmt"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Config string `long:"config" short:"c" description:"config file to use" default:"config.yaml"`
}

var opts = Options{}

func (o *Options) Execute(args []string) error {
	return fmt.Errorf("Main not yet implemented")
}

var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)

func Execute() error {
	if _, err := parser.Parse(); err != nil {
		if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
			fmt.Printf("%s", ferr.Message)
			return nil
		}

		return err
	}

	return nil
}

func main() {

	if err := Execute(); err != nil {
		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}

}
