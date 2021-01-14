package main

import (
	"fmt"
	"os"
	"os/signal"

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

	configPath := "/etc/default/zeus"
	if len(opts.Args.Config) != 0 {
		configPath = string(opts.Args.Config)
	}

	config, err := OpenConfig(configPath)
	if err != nil {
		return err
	}

	z, err := OpenZeus(*config)
	if err != nil {
		return err
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		z.shutdown()
	}()

	return z.run()
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[zeus] Unhandled error: %s\n", err)
		os.Exit(1)
	}
}
