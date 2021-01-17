package main

import (
	"io"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type InterpolationManager struct {
	name         string
	interpoler   zeus.ClimateInterpoler
	capabilities []capability
	reports      chan<- zeus.StateReport
	log          *log.Logger
	period       time.Duration
}

func (i *InterpolationManager) SendState(s zeus.State) {
	for _, c := range i.capabilities {
		c.Action(s)
	}
}

func (i *InterpolationManager) StateReport(current, next zeus.Interpolation, now time.Time, nextTime time.Time) zeus.StateReport {
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

func (i *InterpolationManager) Interpolate(wg *sync.WaitGroup, init, quit <-chan struct{}) {
	defer func() {
		if i.reports != nil {
			close(i.reports)
		}
		wg.Done()
	}()
	i.log.Printf("Starting interpolation loop ")
	<-init
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
	log.Printf("%+v", quit)
	for {
		select {
		case <-quit:
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

func NewInterpolationManager(name string,
	states []zeus.State,
	transitions []zeus.Transition,
	caps []capability,
	reports chan<- zeus.StateReport,
	logs io.Writer) (*InterpolationManager, error) {
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

	return &InterpolationManager{
		name:         path.Join(hostname, "zone", name),
		interpoler:   i,
		capabilities: caps,
		log:          logger,
		reports:      reports,
		period:       5 * time.Second,
	}, nil

}
