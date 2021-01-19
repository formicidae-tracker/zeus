package main

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
