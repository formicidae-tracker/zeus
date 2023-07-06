package main

import "github.com/formicidae-tracker/zeus/internal/zeus"

type TargetReporter interface {
	Reporter
	TargetChannel() chan<- zeus.ClimateTarget
}
