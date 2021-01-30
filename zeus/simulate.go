package main

import (
	"os"
	"os/signal"

	"github.com/formicidae-tracker/zeus"
	flags "github.com/jessevdk/go-flags"
)

type SimulateCommand struct {
	Args struct {
		Hostname   string
		SeasonFile flags.Filename
	} `positional-args:"yes" required:"yes"`

	TimeRatio      float64 `long:"time-ratio" description:"time ratio for the simulation" default:"1.0"`
	OlympusAddress string  `long:"olympus" description:"olympus address to connect to" default:"localhost:3001"`
	RPCPort        int     `long:"rpc-port" description:"the rpc port to use" default:"5011"`
}

func (c *SimulateCommand) Execute(args []string) error {

	season, err := zeus.ReadSeasonFile(string(c.Args.SeasonFile), os.Stderr)
	if err != nil {
		return err
	}

	s, err := NewZeusSimulator(ZeusSimulatorArgs{
		hostname:       c.Args.Hostname,
		season:         *season,
		timeRatio:      c.TimeRatio,
		olympusAddress: c.OlympusAddress,
		rpcPort:        c.RPCPort,
	})
	if err != nil {
		return err
	}

	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt)

	<-sigint

	return s.shutdown()
}

func init() {
	_, err := parser.AddCommand("simulate-climate-control",
		"simulate climate control",
		"simulate climate control from this computer. Will connect to an olympus host and generate stub climate and alarm data",
		&SimulateCommand{})
	if err != nil {
		panic(err.Error())
	}
}
