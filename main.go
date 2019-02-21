package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
	yaml "gopkg.in/yaml.v2"
)

type Options struct {
	Config string `long:"config" short:"c" description:"config file to use" default:"config.yaml"`
}

var opts = Options{}

func (o Options) LoadConfig() (*Config, error) {
	c := &Config{}
	f, err := os.Open(opts.Config)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)

func main() {
	if _, err := parser.Parse(); err != nil {
		if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
			fmt.Printf("%s\n", ferr.Message)
			os.Exit(0)
		}

		log.Printf("Unhandled error: %s", err)
		os.Exit(1)
	}
}
