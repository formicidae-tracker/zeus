package main

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type SimulateCommand struct {
	StartTime string `long:"start-time" short:"s" description:"starting hours and minutes of the simulation like 15:04, using current time if left blank"`
	Duration  int    `long:"duration" short:"d" description:"length of the simulation in days" default:"7"`
}

var simulateCommand = SimulateCommand{}

func (s *SimulateCommand) Execute(args []string) error {
	c, err := opts.LoadConfig()
	if err != nil {
		return err
	}

	var start time.Time
	if len(s.StartTime) == 0 {
		start = time.Now()
	} else {
		var err error
		start, err = time.Parse("15:04", s.StartTime)
		if err != nil {
			return err
		}
		y, m, d := time.Now().Date()
		start = start.AddDate(y, int(m), d)
	}

	for n, z := range c.Zones {
		fmt.Printf("=== Simulating zone '%s' for %d day from %s ===\n", n, s.Duration, start.Format("Mon Jan 02 15:04:05 -0700 MST 2006"))

		i, err := zeus.NewClimateInterpoler(z.States, z.Transitions, start.UTC())
		if err != nil {
			return err
		}
		var t time.Time
		for t = start; t.Before(start.AddDate(0, 0, s.Duration)); {
			toTest := t.Add(1 * time.Second)
			inter, next, nextInterpolation := i.CurrentInterpolation(toTest)
			fmt.Printf("%s state is %s\n", t.Local().Format("Mon Jan 02 15:04:05 -0700 MST 2006"), inter)
			if nextInterpolation == nil {
				fmt.Printf("No more transition\n")
				t = start.AddDate(0, 0, s.Duration)
				continue
			}
			t = next
		}
		fmt.Printf("=== End of simulation at %s ===\n", t.Format("Mon Jan 02 15:04:05 -0700 MST 2006"))
	}

	return nil
}

func init() {
	parser.AddCommand("simulate",
		"computes and display the climate states",
		"simulate the climate states and transitions and display it on stdout by default",
		&simulateCommand)
}
