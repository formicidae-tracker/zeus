package main

import (
	"github.com/formicidae-tracker/zeus"
)

type ZoneClimateRunner interface {
	Run()
	Close() error
}

type zoneClimateRunner struct {
	interpoler Interpoler

	capabilities []capability
	alarmMonitor AlarmMonitor

	reporter *RPCReporter
}

func (r *zoneClimateRunner) Run() {
}

func (r *zoneClimateRunner) Close() error {
	return nil
}

func NewZoneClimateRunner(d ArkeDispatcher, definition ZoneDefinition, climate zeus.ZoneClimate, olympusHost string) (ZoneClimateRunner, error) {
	return &zoneClimateRunner{}, nil

}
