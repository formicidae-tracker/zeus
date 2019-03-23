package main

import (
	"io"
	"log"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"github.com/formicidae-tracker/dieu"
)

type InterpolationManager struct {
	name         string
	interpoler   ClimateInterpoler
	capabilities []capability
	reports      chan<- dieu.StateReport
	log          *log.Logger
	period       time.Duration
}

func (i *InterpolationManager) SendState(s dieu.State) {
	for _, c := range i.capabilities {
		c.Action(s)
	}
}

func sanitizeUnit(u dieu.BoundedUnit) float64 {
	if dieu.IsUndefined(u) || math.IsNaN(u.Value()) {
		return -1000.0
	}
	return u.Value()
}

func sanitizeState(s dieu.State) dieu.State {
	return dieu.State{
		Name:         s.Name,
		Temperature:  dieu.Temperature(sanitizeUnit(s.Temperature)),
		Humidity:     dieu.Humidity(sanitizeUnit(s.Humidity)),
		Wind:         dieu.Wind(sanitizeUnit(s.Wind)),
		VisibleLight: dieu.Light(sanitizeUnit(s.VisibleLight)),
		UVLight:      dieu.Light(sanitizeUnit(s.UVLight)),
	}
}

func (i *InterpolationManager) StateReport(current, next Interpolation, now time.Time, nextTime time.Time) dieu.StateReport {
	report := dieu.StateReport{
		Zone:       i.name,
		Current:    sanitizeState(current.State(now)),
		CurrentEnd: nil,

		NextTime: nil,
		Next:     nil,
		NextEnd:  nil,
	}
	if inter, ok := current.(*transition); ok == true {
		report.CurrentEnd = &dieu.State{}
		*report.CurrentEnd = sanitizeState(inter.State(now.Add(inter.duration)))
	}

	if next != nil {
		report.NextTime = &time.Time{}
		*report.NextTime = nextTime
		report.Next = &dieu.State{}
		*report.Next = sanitizeState(next.State(nextTime))
		if inter, ok := next.(*transition); ok == true {
			report.NextEnd = &dieu.State{}
			*report.NextEnd = sanitizeState(inter.State(nextTime.Add(inter.duration)))
		}

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

	_, isTransition := cur.(*transition)

	if i.reports != nil {
		report := i.StateReport(cur, next, now, nextTime)
		i.reports <- report
	}

	timer := time.NewTicker(i.period)
	defer timer.Stop()

	for {
		select {
		case <-quit:
			i.log.Printf("Closing climate interpolation")
			return
		case now := <-timer.C:
			new, nextTime, next := i.interpoler.CurrentInterpolation(now)
			_, newIsTransition := new.(*transition)

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
	states []dieu.State,
	transitions []dieu.Transition,
	caps []capability,
	reports chan<- dieu.StateReport,
	logs io.Writer) (*InterpolationManager, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(logs, "[zone/"+name+"/climate]: ", log.LstdFlags)
	logger.Printf("Computing climate interpolation")
	i, err := NewClimateInterpoler(states, transitions, time.Now().UTC())
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
