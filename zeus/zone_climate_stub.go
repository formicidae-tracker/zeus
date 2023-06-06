package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
	"github.com/formicidae-tracker/zeus/zeuspb"
)

type stubAlarm struct {
	Alarm  zeus.Alarm
	Period time.Duration
	Next   time.Time
	On     bool
}

type zoneClimateStub struct {
	host, zone  string
	timeRatio   float64
	rpcReporter *RPCReporter

	interpoler    zeus.ClimateInterpoler
	current, next zeus.Interpolation

	stubAlarms []*stubAlarm
	alarms     []zeus.AlarmEvent
	reports    []zeus.ClimateReport

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
		host:           args.hostname,
		zone:           args.zoneName,
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
	ready := make(chan struct{})
	go func() { s.rpcReporter.Report(ready); close(s.done) }()
	defer func() {
		close(s.rpcReporter.ReportChannel())
		close(s.rpcReporter.TargetChannel())
		close(s.rpcReporter.AlarmChannel())
	}()

	now := time.Now()
	basePeriod := 500 * time.Millisecond
	period := time.Duration(basePeriod.Seconds()*1000.0/s.timeRatio) * time.Millisecond
	ticker := time.NewTicker(period)

	s.stubAlarms = []*stubAlarm{
		&stubAlarm{
			Alarm:  zeus.WaterLevelCritical,
			Period: 20 * time.Minute,
			Next:   now.Add(3 * time.Minute),
			On:     false,
		},
		&stubAlarm{
			Alarm:  zeus.HumidityUnreachable,
			Period: 30 * time.Minute,
			Next:   now.Add(1 * time.Minute),
			On:     false,
		},
	}

	s.step(now)

	defer ticker.Stop()
	<-ready
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now = now.Add(basePeriod)
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

func (s *zoneClimateStub) Last() *zeuspb.ZoneStatus {
	return &zeuspb.ZoneStatus{}
}

func (s *zoneClimateStub) step(now time.Time) {
	s.simulateClimate(now)
	s.simulateAlarms(now)
}

func (s *zoneClimateStub) simulateClimate(now time.Time) {
	new, nextTime, next := s.interpoler.CurrentInterpolation(now)
	s.next = next
	sendReport := false
	if s.current == nil || s.current.String() != new.String() {
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

func (s *zoneClimateStub) sendAlarm(ae zeus.AlarmEvent) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.rpcReporter.AlarmChannel() <- ae
	ae.ZoneIdentifier = ""
	s.alarms = append(s.alarms, ae)
}

func (s *zoneClimateStub) simulateAlarms(now time.Time) {
	for _, a := range s.stubAlarms {
		if now.Before(a.Next) {
			continue
		}
		ae := zeus.AlarmEvent{
			Flags:          a.Alarm.Flags(),
			Identifier:     a.Alarm.Identifier(),
			Description:    a.Alarm.Description(),
			ZoneIdentifier: zeus.ZoneIdentifier(s.host, s.zone),
			Time:           now,
		}
		a.Next = a.Next.Add(a.Period + time.Duration(rand.NormFloat64()*a.Period.Seconds())*time.Second)

		if a.On == true {
			a.On = false
			ae.Status = zeus.AlarmOff
		} else {
			a.On = true
			ae.Status = zeus.AlarmOn
		}
		s.sendAlarm(ae)
	}
}

func (s *zoneClimateStub) sendState(state zeus.State, now time.Time) {
	noData := true
	cr := zeus.ClimateReport{
		Humidity:     zeus.UndefinedHumidity,
		Temperatures: []zeus.Temperature{zeus.UndefinedTemperature},
		Time:         now,
	}
	if zeus.IsUndefined(state.Humidity) == false {
		cr.Humidity = zeus.Humidity(state.Humidity.Value() + rand.NormFloat64()*1.0)
		noData = false
	}
	if zeus.IsUndefined(state.Temperature) == false {
		cr.Temperatures[0] = zeus.Temperature(state.Temperature.Value() + rand.NormFloat64()*0.01)
		noData = false
	}

	if noData == true {
		return
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	s.rpcReporter.ReportChannel() <- cr
	s.reports = append(s.reports, cr)
}

func (s *zoneClimateStub) sendReport(now, next time.Time) {
	target := zeus.ClimateTarget{
		ZoneIdentifier: zeus.ZoneIdentifier(s.host, s.zone),
		Current:        s.current.State(now),
		CurrentEnd:     s.current.End(),
	}
	if s.next != nil {
		target.NextTime = &time.Time{}
		*target.NextTime = next
		target.Next = &zeus.State{}
		*target.Next = s.next.State(next)
		target.NextEnd = s.next.End()
	}
	s.rpcReporter.TargetChannel() <- target
}
