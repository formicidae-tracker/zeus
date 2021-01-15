package main

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/zeus"
	"github.com/jessevdk/go-flags"
)

type SimulateCommand struct {
	StartTime string `long:"start-time" short:"s" description:"starting hours and minutes of the simulation like 15:04, using current time if left blank"`
	Duration  int    `long:"duration" short:"d" description:"length of the simulation in days" default:"7"`

	Args struct {
		SeasonFile flags.Filename
	} `positional-args:"yes" required:"true"`
}

var simulateCommand = &SimulateCommand{}

func (c *SimulateCommand) Execute(args []string) error {
	season, err := zeus.ReadSeasonFile(string(c.Args.SeasonFile))
	if err != nil {
		return err
	}

	var start time.Time
	if len(c.StartTime) == 0 {
		start = time.Now()
	} else {
		var err error
		start, err = time.Parse("15:04", c.StartTime)
		if err != nil {
			return err
		}
		y, m, d := time.Now().Date()
		start = start.AddDate(y, int(m), d)
	}

	for name, zone := range season.Zones {
		fmt.Printf("=== Simulating zone '%s' for %d day from %s ===\n", name, c.Duration, start.Format("Mon Jan 02 15:04:05 -0700 MST 2006"))

		i, err := zeus.NewClimateInterpoler(zone.States, zone.Transitions, start.UTC())
		if err != nil {
			return err
		}
		var t time.Time
		for t = start; t.Before(start.AddDate(0, 0, c.Duration)); {
			toTest := t.Add(1 * time.Second)
			inter, next, nextInterpolation := i.CurrentInterpolation(toTest)
			fmt.Printf("%s state is %s\n", t.Local().Format("Mon Jan 02 15:04:05 -0700 MST 2006"), inter)
			if nextInterpolation == nil {
				fmt.Printf("No more transition\n")
				t = start.AddDate(0, 0, c.Duration)
				continue
			}
			t = next
		}
		fmt.Printf("=== End of simulation at %s ===\n", t.Format("Mon Jan 02 15:04:05 -0700 MST 2006"))
	}

	return nil
}

func init() {
	_, err := parser.AddCommand("simulate",
		"simulate a season file",
		"simulate a season file, and displays climate states and transitions on stdout",
		simulateCommand)
	if err != nil {
		panic(err.Error())
	}
}
