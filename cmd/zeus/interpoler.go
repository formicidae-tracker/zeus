package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/formicidae-tracker/zeus/internal/zeus"
)

type Interpoler interface {
	Interpolate(chan<- struct{})

	States() <-chan zeus.State
	Reports() <-chan zeus.ClimateTarget

	Close() error
}

type interpoler struct {
	Period time.Duration

	logger     *log.Logger
	name       string
	interpoler zeus.ClimateInterpoler
	quit       chan struct{}
	states     chan zeus.State
	reports    chan zeus.ClimateTarget
}

func (i *interpoler) stateReport(current, next zeus.Interpolation, now time.Time, nextTime time.Time) zeus.ClimateTarget {
	report := zeus.ClimateTarget{
		ZoneIdentifier: i.name,
		Current:        current.State(now),
		CurrentEnd:     nil,

		NextTime: nil,
		Next:     nil,
		NextEnd:  nil,
	}
	report.CurrentEnd = current.End()

	if next != nil {
		report.NextTime = &time.Time{}
		*report.NextTime = nextTime
		report.Next = &zeus.State{}
		*report.Next = next.State(nextTime)
		report.NextEnd = next.End()
	}
	return report
}

func (i *interpoler) sendReport(r zeus.ClimateTarget) {
	select {
	case i.reports <- r:
	default:
	}
}

func (i *interpoler) sendState(s zeus.State) {
	i.states <- s
}

func (i *interpoler) Interpolate(ready chan<- struct{}) {
	i.quit = make(chan struct{})
	defer func() {
		close(i.states)
		close(i.reports)
	}()

	i.logger.Printf("Starting interpolation loop")
	now := time.Now()
	cur, nextTime, next := i.interpoler.CurrentInterpolation(now)
	i.logger.Printf("Starting interpolation is %s", cur)

	i.sendState(cur.State(now))

	isTransition := cur.End() != nil

	i.sendReport(i.stateReport(cur, next, now, nextTime))

	timer := time.NewTicker(i.Period)
	defer timer.Stop()

	close(ready)
	for {
		select {
		case <-i.quit:
			i.logger.Printf("Closing climate interpolation")
			return
		case now := <-timer.C:
			new, nextTime, next := i.interpoler.CurrentInterpolation(now)
			newIsTransition := new.End() != nil

			if isTransition != newIsTransition {
				i.logger.Printf("New interpolation %s", new)
				cur = new
				isTransition = newIsTransition
			} else if isTransition == false {
				i.sendState(cur.State(now))
				continue
			}
			s := cur.State(now)
			i.sendState(s)
			report := i.stateReport(cur, next, now, nextTime)
			i.sendReport(report)
		}
	}
}

func NewInterpoler(name string, states []zeus.State, transitions []zeus.Transition) (Interpoler, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stderr, "[zone/"+name+"/climate]: ", 0)
	i, err := zeus.NewClimateInterpoler(states, transitions, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return &interpoler{
		name:       path.Join(hostname, "zone", name),
		interpoler: i,
		logger:     logger,
		reports:    make(chan zeus.ClimateTarget, 1),
		states:     make(chan zeus.State, 1),
		Period:     5 * time.Second,
	}, nil
}

func (i *interpoler) States() <-chan zeus.State {
	return i.states
}

func (i *interpoler) Reports() <-chan zeus.ClimateTarget {
	return i.reports
}

func (i *interpoler) Close() error {
	if i.quit == nil {
		return fmt.Errorf("Already closed")
	}
	close(i.quit)
	i.quit = nil
	return nil
}
