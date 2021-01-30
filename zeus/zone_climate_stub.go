package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type stubAlarm struct {
	Alarm zeus.Alarm
	Last  time.Time
	On    bool
}

type zoneClimateStub struct {
	host, zone  string
	timeRatio   float64
	rpcReporter *RPCReporter

	interpoler    zeus.ClimateInterpoler
	current, next zeus.Interpolation

	alarms  []zeus.AlarmEvent
	reports []zeus.ClimateReport

	done, stop chan struct{}
	mx         sync.Mutex
}

type ZoneClimateStubArgs struct {
	hostname       string
	zoneName       string
	climate        zeus.ZoneClimate
	timeRatio      float64
	olympusAddress string
	rpcPort        int
}

func NewZoneClimateStub(args ZoneClimateStubArgs) (ZoneClimateRunner, error) {
	res := &zoneClimateStub{
		host:      args.hostname,
		zone:      args.zoneName,
		timeRatio: args.timeRatio,
	}
	var err error
	res.interpoler, err = zeus.NewClimateInterpoler(args.climate.States, args.climate.Transitions, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	res.rpcReporter, err = NewRPCReporter(RPCReporterOptions{
		wantedHostname: args.hostname,
		zoneName:       args.zoneName,
		climate:        args.climate,
		olympusAddress: args.olympusAddress,
		rpcPort:        args.rpcPort,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *zoneClimateStub) Run() {
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	defer close(s.done)

	now := time.Now()
	period := time.Duration(500.0/s.timeRatio) * time.Millisecond
	ticker := time.NewTicker(period)
	s.step(now)

	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now = now.Add(period)
			s.step(now)
		}
	}
}

func (s *zoneClimateStub) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("already closed")
		}
		<-s.done
	}()
	close(s.stop)
	return nil
}

func (s *zoneClimateStub) ClimateLog(start, end int) ([]zeus.ClimateReport, error) {
	err := checkRange(start, end)
	if err != nil {
		return nil, err
	}

	s.mx.Lock()
	defer s.mx.Unlock()
	start, end, err = clampRange(start, end, len(s.reports))
	if err != nil {
		return nil, err
	}
	res := make([]zeus.ClimateReport, end-start)
	copy(res, s.reports[start:end])

	return res, nil
}

func (s *zoneClimateStub) AlarmLog(start, end int) ([]zeus.AlarmEvent, error) {
	err := checkRange(start, end)
	if err != nil {
		return nil, err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	start, end, err = clampRange(start, end, len(s.alarms))
	if err != nil {
		return nil, err
	}
	res := make([]zeus.AlarmEvent, end-start)
	copy(res, s.alarms[start:end])

	return res, nil
}

func (s *zoneClimateStub) step(now time.Time) {
	s.simulateClimate(now)
	s.simulateAlarms(now)
}

func (s *zoneClimateStub) simulateClimate(now time.Time) {
	new, nextTime, next := s.interpoler.CurrentInterpolation(now)
	s.next = next
	sendReport := false
	if s.current.String() != new.String() {
		s.current = new
		sendReport = true
	}
	if s.current.End() != nil {
		sendReport = true
	}
	s.sendState(s.current.State(now), now)
	if sendReport == true {
		s.sendReport(now, nextTime)
	}
}

func (s *zoneClimateStub) simulateAlarms(now time.Time) {
}

func (s *zoneClimateStub) sendState(state zeus.State, now time.Time) {
	cr := zeus.ClimateReport{
		Humidity:     zeus.Humidity(state.Humidity.Value() * rand.NormFloat64() * 1.0),
		Temperatures: []zeus.Temperature{zeus.Temperature(state.Temperature.Value() * rand.NormFloat64() * 0.01)},
		Time:         now,
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	s.rpcReporter.ReportChannel() <- cr
	s.reports = append(s.reports, cr)
}

func (s *zoneClimateStub) sendReport(now, next time.Time) {
	report := zeus.StateReport{
		ZoneIdentifier: zeus.ZoneIdentifier(s.host, s.zone),
		Current:        zeus.SanitizeState(s.current.State(now)),
		CurrentEnd:     s.current.End(),
	}
	if s.next != nil {
		report.NextTime = &time.Time{}
		*report.NextTime = next
		report.Next = &zeus.State{}
		*report.Next = zeus.SanitizeState(s.next.State(next))
		report.NextEnd = s.next.End()
	}
	s.rpcReporter.StateChannel() <- report
}
