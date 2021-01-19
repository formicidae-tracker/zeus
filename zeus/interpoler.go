package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type Interpoler struct {
	name         string
	interpoler   zeus.ClimateInterpoler
	capabilities []capability
	reports      chan<- zeus.StateReport
	log          *log.Logger
	period       time.Duration
	quit         chan struct{}
}

func (i *Interpoler) SendState(s zeus.State) {
	for _, c := range i.capabilities {
		c.Action(s)
	}
}

func (i *Interpoler) StateReport(current, next zeus.Interpolation, now time.Time, nextTime time.Time) zeus.StateReport {
	report := zeus.StateReport{
		Zone:       i.name,
		Current:    zeus.SanitizeState(current.State(now)),
		CurrentEnd: nil,

		NextTime: nil,
		Next:     nil,
		NextEnd:  nil,
	}
	report.CurrentEnd = current.End()

	if next != nil {
		report.NextTime = &time.Time{}
		*report.NextTime = nextTime
		report.Next = &zeus.State{}
		*report.Next = zeus.SanitizeState(next.State(nextTime))
		report.NextEnd = next.End()
	}
	return report
}

func (i *Interpoler) Interpolate() {
	defer func() {
		if i.reports != nil {
			close(i.reports)
		}
	}()
	i.quit = make(chan struct{})

	i.log.Printf("Starting interpolation loop ")
	now := time.Now()
	cur, nextTime, next := i.interpoler.CurrentInterpolation(now)
	i.log.Printf("Starting interpolation is %s", cur)

	i.SendState(cur.State(now))

	isTransition := cur.End() != nil

	if i.reports != nil {
		report := i.StateReport(cur, next, now, nextTime)
		i.reports <- report
	}

	timer := time.NewTicker(i.period)
	defer timer.Stop()

	for {
		select {
		case <-i.quit:
			i.log.Printf("Closing climate interpolation")
			return
		case now := <-timer.C:
			new, nextTime, next := i.interpoler.CurrentInterpolation(now)
			newIsTransition := new.End() != nil

			if isTransition != newIsTransition {
				i.log.Printf("New interpolation %s", new)
				cur = new
				isTransition = newIsTransition
			} else if isTransition == false {
				i.SendState(cur.State(now))
				continue
			}
			s := cur.State(now)
			i.SendState(s)
			if i.reports != nil {
				report := i.StateReport(cur, next, now, nextTime)
				i.reports <- report
			}
		}
	}
}

func NewInterpoler(name string,
	states []zeus.State,
	transitions []zeus.Transition,
	caps []capability,
	reports chan<- zeus.StateReport,
	logs io.Writer) (*Interpoler, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(logs, "[zone/"+name+"/climate]: ", 0)
	logger.Printf("Computing climate interpolation")
	i, err := zeus.NewClimateInterpoler(states, transitions, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return &Interpoler{
		name:         path.Join(hostname, "zone", name),
		interpoler:   i,
		capabilities: caps,
		log:          logger,
		reports:      reports,
		period:       5 * time.Second,
	}, nil

}

func (i *Interpoler) Close() error {
	if i.quit == nil {
		return fmt.Errorf("Already closed")
	}
	close(i.quit)
	i.quit = nil
	return nil
}
