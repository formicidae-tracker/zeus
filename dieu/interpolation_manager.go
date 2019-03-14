package main

import (
	"log"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"git.tuleu.science/fort/dieu"
)

type InterpolationManager struct {
	name         string
	interpoler   ClimateInterpoler
	capabilities []capability
	reports      chan<- dieu.StateReport
	log          *log.Logger
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

func (i *InterpolationManager) StateReport(s dieu.State, t time.Time) dieu.StateReport {
	report := dieu.StateReport{
		Zone:     i.name,
		Current:  sanitizeState(s),
		NextTime: nil,
		Next:     nil,
	}
	if nextT, ok := i.interpoler.NextInterpolationTime(t); ok == true {
		report.NextTime = &time.Time{}
		*report.NextTime = nextT
		report.Next = &dieu.State{}
		*report.Next = sanitizeState(i.interpoler.CurrentInterpolation(nextT.Add(1 * time.Second)).State(nextT))
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
	cur := i.interpoler.CurrentInterpolation(now)
	i.log.Printf("Starting interpolation is %s", cur)

	i.SendState(cur.State(now))

	if i.reports != nil {
		report := i.StateReport(cur.State(now), now)
		i.reports <- report
	}

	timer := time.NewTicker(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-quit:
			i.log.Printf("Closing climate interpolation")
			return
		case <-timer.C:
			now := time.Now()
			new := i.interpoler.CurrentInterpolation(now)
			if cur == new {
				continue
			}
			cur = new
			s := cur.State(now)
			i.log.Printf("New interpolation %s", new)
			i.SendState(s)
			if i.reports != nil {
				report := i.StateReport(s, now)
				i.reports <- report
			}
		}
	}
}

func NewInterpolationManager(name string,
	states []dieu.State,
	transitions []dieu.Transition,
	caps []capability,
	reports chan<- dieu.StateReport) (*InterpolationManager, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stderr, "[zone/"+name+"/climate]: ", log.LstdFlags)
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
	}, nil

}
