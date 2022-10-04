package main

import (
	"fmt"
	"io/ioutil"

	"github.com/formicidae-tracker/zeus"
	"github.com/jessevdk/go-flags"
)

type StartCommand struct {
	Args struct {
		Node       Nodename
		SeasonFile flags.Filename
	} `positional-args:"yes" required:"yes"`
}

func (c *StartCommand) Execute(args []string) error {
	seasonContent, err := ioutil.ReadFile(string(c.Args.SeasonFile))
	if err != nil {
		return fmt.Errorf("could not read '%s': %w", c.Args.SeasonFile, err)
	}
	_, err = zeus.ParseSeasonFile(seasonContent)
	if err != nil {
		return fmt.Errorf("invalid season file: %w", err)
	}

	node, err := GetNode(c.Args.Node)
	if err != nil {
		return err
	}

	return node.StartClimate(seasonContent)
}

type StopCommand struct {
	Args struct {
		Node Nodename
	} `positional-args:"yes" required:"yes"`
}

func (c *StopCommand) Execute(args []string) error {
	node, err := GetNode(c.Args.Node)
	if err != nil {
		return err
	}
	return node.StopClimate()
}

func init() {
	_, err := parser.AddCommand("start",
		"starts climate on node",
		"starts a climate on a specified node",
		&StartCommand{})
	if err != nil {
		panic(err.Error())
	}

	_, err = parser.AddCommand("stop",
		"stops climate on node",
		"stops climate on a specified node",
		&StopCommand{})
	if err != nil {
		panic(err.Error())
	}

}
