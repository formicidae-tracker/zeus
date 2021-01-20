package main

import "github.com/formicidae-tracker/zeus"

type StateReporter interface {
	Reporter
	StateChannel() chan<- zeus.StateReport
}
