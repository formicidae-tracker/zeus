package main

import "fmt"

type SimulateCommand struct {
}

var simulateCommand = SimulateCommand{}

func (s *SimulateCommand) Execute(args []string) error {
	_, err := opts.LoadConfig()
	if err != nil {
		return err
	}

	return fmt.Errorf("Simulate not implemented")
}

func init() {
	parser.AddCommand("simulate",
		"computes and display the climate states",
		"simulate the climate states and transitions and display it on stdout by default",
		&simulateCommand)
}
