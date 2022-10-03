package main

import "github.com/formicidae-tracker/zeus"

type lastStateReporter struct {
	requests chan chan zeus.ZeusZoneStatus

	states   chan zeus.ClimateTarget
	climates chan zeus.ClimateReport

	last zeus.ZeusZoneStatus
}

func (r *lastStateReporter) Report(ready chan<- struct{}) {
	defer close(r.requests)
	close(ready)
	for {
		select {
		case s, ok := <-r.states:
			if ok == false {
				r.states = nil
			} else {
				r.last.State = s.Current
			}
		case report, ok := <-r.climates:
			if ok == false {
				r.climates = nil
			} else {
				r.last.Humidity = report.Humidity.Value()
				if len(report.Temperatures) > 0 {
					r.last.Temperature = report.Temperatures[0].Value()
				}
			}
		case req := <-r.requests:
			req <- zeus.ZeusZoneStatus{
				Temperature: r.last.Temperature,
				Humidity:    r.last.Humidity,
				State:       r.last.State,
			}
		}

		if r.states == nil && r.climates == nil {
			return
		}
	}
}

func NewLastStateReporter() *lastStateReporter {
	return &lastStateReporter{
		requests: make(chan chan zeus.ZeusZoneStatus),
		climates: make(chan zeus.ClimateReport, 10),
		states:   make(chan zeus.ClimateTarget, 1),
	}
}

func (r *lastStateReporter) ReportChannel() chan<- zeus.ClimateReport {
	return r.climates
}

func (r *lastStateReporter) TargetChannel() chan<- zeus.ClimateTarget {
	return r.states
}

func (r *lastStateReporter) Last() (res zeus.ZeusZoneStatus) {
	defer func() {
		if recover() != nil {
			res = zeus.ZeusZoneStatus{}
		}
	}()
	resChannel := make(chan zeus.ZeusZoneStatus)
	r.requests <- resChannel
	return <-resChannel
}
