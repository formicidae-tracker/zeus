package main

import "github.com/formicidae-tracker/zeus"

type ZoneClimateRunner struct {
	interpoler   *Interpoler
	reporter     *RPCReporter
	capabilities []capability
	alarmMonitor AlarmMonitor
}

func (r *ZoneClimateRunner) Run() {
}

func (r *ZoneClimateRunner) Stop() {
}

func NewZoneClimateRunner(climate zeus.ZoneClimate,
	definition zone.Definition,
	listener BusListener) (*ZoneClimateRunner, error) {

}
