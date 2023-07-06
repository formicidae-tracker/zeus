package main

import (
	"github.com/barkimedes/go-deepcopy"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/formicidae-tracker/zeus/pkg/zeuspb"
)

type lastStateReporter struct {
	requests chan chan *zeuspb.ZoneStatus

	targets chan zeus.ClimateTarget
	reports chan zeus.ClimateReport

	last zeuspb.ZoneStatus
}

func (r *lastStateReporter) Report(ready chan<- struct{}) {
	defer close(r.requests)
	close(ready)
	for {
		select {
		case target, ok := <-r.targets:
			if ok == false {
				r.targets = nil
			} else {
				r.last.Target = target.Current.AsPbTarget()
			}
		case report, ok := <-r.reports:
			if ok == false {
				r.reports = nil
			} else {
				r.last.Humidity = zeus.AsFloat32Pointer(report.Humidity)
				if len(report.Temperatures) > 0 {
					r.last.Temperature = zeus.AsFloat32Pointer(report.Temperatures[0])
				}
			}
		case req := <-r.requests:
			req <- deepcopy.MustAnything(&r.last).(*zeuspb.ZoneStatus)
		}

		if r.targets == nil && r.reports == nil {
			return
		}
	}
}

func NewLastStateReporter() *lastStateReporter {
	return &lastStateReporter{
		requests: make(chan chan *zeuspb.ZoneStatus),
		reports:  make(chan zeus.ClimateReport, 10),
		targets:  make(chan zeus.ClimateTarget, 1),
	}
}

func (r *lastStateReporter) ReportChannel() chan<- zeus.ClimateReport {
	return r.reports
}

func (r *lastStateReporter) TargetChannel() chan<- zeus.ClimateTarget {
	return r.targets
}

func (r *lastStateReporter) Last() (res *zeuspb.ZoneStatus) {
	defer func() {
		if recover() != nil {
			res = nil
		}
	}()
	resChannel := make(chan *zeuspb.ZoneStatus)
	r.requests <- resChannel
	return <-resChannel
}
