package main

import "github.com/formicidae-tracker/zeus"

type TargetReporter interface {
	Reporter
	TargetChannel() chan<- zeus.ClimateTarget
}
