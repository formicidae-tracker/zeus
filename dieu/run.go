package main

import "fmt"

type RunCommand struct {
}

func (c *RunCommand) Execute(args []string) error {
	return fmt.Errorf("Run command is not yet implemented")
}

var runCommand = RunCommand{}

func init() {
	parser.AddCommand("run",
		"run the climate control",
		"run the climate control on the real hardware",
		&runCommand)
}
