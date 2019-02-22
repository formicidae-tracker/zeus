package main

import (
	"fmt"
	"time"
)

type SimulateCommand struct {
	StartTime time.Time `long:"start-time" short:"s" description:"starting time of the simulation, using current time if left blank"`
	Duration  int       `long:"duration" short:"d" description:"length of the simulation in days" default:"7"`
}

var simulateCommand = SimulateCommand{}

func (s *SimulateCommand) Execute(args []string) error {
	c, err := opts.LoadConfig()
	if err != nil {
		return err
	}

	if s.StartTime.IsZero() == true {
		s.StartTime = time.Now()
	}

	for n, z := range c.Zones {
		fmt.Printf("=== Simulating zone '%s' for %d day from %s ===\n", n, s.Duration, s.StartTime.Format("Mon Jan 02 15:04:05 -0700 MST 2006"))

		i, err := NewClimateInterpoler(z.States, z.Transitions, s.StartTime.UTC())
		if err != nil {
			return err
		}
		var t time.Time
		for t = s.StartTime; t.Before(s.StartTime.AddDate(0, 0, s.Duration)); {
			toTest := t.Add(1 * time.Second)
			inter := i.CurrentInterpolation(toTest)
			fmt.Printf("%s state is %s\n", t.Local().Format("Mon Jan 02 15:04:05 -0700 MST 2006"), inter)
			next, ok := i.NextInterpolationTime(toTest)
			if ok == false {
				fmt.Printf("No more transition\n")
				t = s.StartTime.AddDate(0, 0, s.Duration)
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
