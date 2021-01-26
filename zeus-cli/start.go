package main

import (
	"os"

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
	season, err := zeus.ReadSeasonFile(string(c.Args.SeasonFile), os.Stderr)
	if err != nil {
		return err
	}
	node, err := GetNode(c.Args.Node)
	if err != nil {
		return err
	}
	unused := 0
	return node.RunMethod("Zeus.StartClimate",
		zeus.ZeusStartArgs{
			Version: zeus.ZEUS_VERSION,
			Season:  *season,
		}, &unused)
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
	unused := 0
	return node.RunMethod("Zeus.StopClimate", 0, &unused)
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
