package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/formicidae-tracker/olympus/olympuspb"
	"github.com/formicidae-tracker/zeus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPCReporter struct {
	declaration *olympuspb.ZoneDeclaration
	addr        string
	lastReport  *olympuspb.ClimateReport
	lastTarget  *olympuspb.ClimateTarget

	runner ZoneClimateRunner

	climateReports chan zeus.ClimateReport
	alarmReports   chan zeus.AlarmEvent
	climateTargets chan zeus.ClimateTarget
	connected      chan bool

	log *log.Logger
}

func (r *RPCReporter) ReportChannel() chan<- zeus.ClimateReport {
	return r.climateReports
}

func (r *RPCReporter) AlarmChannel() chan<- zeus.AlarmEvent {
	return r.alarmReports
}

func (r *RPCReporter) TargetChannel() chan<- zeus.ClimateTarget {
	return r.climateTargets
}

func buildBackLog(reports []zeus.ClimateReport, events []zeus.AlarmEvent) *olympuspb.ZoneUpStream {
	res := &olympuspb.ZoneUpStream{
		Reports: make([]*olympuspb.ClimateReport, len(reports)),
		Alarms:  make([]*olympuspb.AlarmEvent, len(events)),
	}
	for i, r := range reports {
		res.Reports[i] = buildOlympusClimateReport(r)
	}
	for i, e := range events {
		res.Alarms[i] = buildOlympusAlarmEvent(e)
	}
	return res
}

func buildOlympusClimateReport(report zeus.ClimateReport) *olympuspb.ClimateReport {
	temperatures := make([]float32, len(report.Temperatures))
	for i, t := range report.Temperatures {
		temperatures[i] = float32(t)
	}
	return &olympuspb.ClimateReport{
		Time:         timestamppb.New(report.Time),
		Humidity:     zeus.AsFloat32Pointer(report.Humidity),
		Temperatures: temperatures,
	}
}

func buildOlympusAlarmEvent(event zeus.AlarmEvent) *olympuspb.AlarmEvent {
	status := olympuspb.AlarmStatus_ON
	if event.Status == zeus.AlarmOff {
		status = olympuspb.AlarmStatus_OFF
	}
	level := olympuspb.AlarmLevel_WARNING
	if event.Flags&zeus.Emergency != 0x00 {
		level = olympuspb.AlarmLevel_EMERGENCY
	}
	return &olympuspb.AlarmEvent{
		Reason: event.Reason,
		Status: status,
		Time:   timestamppb.New(event.Time),
		Level:  level,
	}
}

func buildOlympusState(s *zeus.State) *olympuspb.ClimateState {
	if s == nil {
		return nil
	}
	return &olympuspb.ClimateState{
		Name:         s.Name,
		Temperature:  zeus.AsFloat32Pointer(s.Temperature),
		Humidity:     zeus.AsFloat32Pointer(s.Humidity),
		Wind:         zeus.AsFloat32Pointer(s.Wind),
		VisibleLight: zeus.AsFloat32Pointer(s.VisibleLight),
		UvLight:      zeus.AsFloat32Pointer(s.UVLight),
	}
}

func buildOlympusClimateTarget(target zeus.ClimateTarget) *olympuspb.ClimateTarget {
	res := &olympuspb.ClimateTarget{
		Current:    buildOlympusState(&target.Current),
		CurrentEnd: buildOlympusState(target.CurrentEnd),
		Next:       buildOlympusState(target.Next),
		NextEnd:    buildOlympusState(target.NextEnd),
	}

	if target.NextTime != nil {
		res.NextTime = timestamppb.New(*target.NextTime)
	}

	return res
}

func paginateAsync(c context.Context,
	ch chan<- *olympuspb.ZoneUpStream,
	pageSize int,
	reports []zeus.ClimateReport,
	events []zeus.AlarmEvent) {

	defer func() {
		res := recover()
		if res != nil && res != 1 {
			panic(res)
		}
		close(ch)
	}()

	push := func(m *olympuspb.ZoneUpStream) {
		select {
		case <-c.Done():
			panic(1)
		case ch <- m:
			time.Sleep(50 * time.Millisecond)
		}
	}

	if pageSize == 0 {
		push(buildBackLog(reports, events))
		return
	}
	for i := 0; i < len(reports); i += pageSize {
		end := i + pageSize
		if end > len(reports) {
			end = len(reports)
		}
		push(buildBackLog(reports[i:end], nil))
	}
	for i := 0; i < len(events); i += pageSize {
		end := i + pageSize
		if end > len(events) {
			end = len(events)
		}
		push(buildBackLog(nil, events[i:end]))

	}
}

func (r *RPCReporter) paginateBacklogs(c context.Context, confirmation *olympuspb.ZoneRegistrationConfirmation) <-chan *olympuspb.ZoneUpStream {
	if confirmation == nil || r.runner == nil {
		return nil
	}
	climateLog, err := r.runner.ClimateLog(0, 0)
	if err != nil {
		r.log.Printf("could not get climate log: %s", err)
	}

	eventLog, err := r.runner.AlarmLog(0, 0)
	if err != nil {
		r.log.Printf("could not get alarm log")
	}
	if eventLog == nil && climateLog == nil {
		return nil
	}
	res := make(chan *olympuspb.ZoneUpStream)
	go paginateAsync(c, res, int(confirmation.PageSize), climateLog, eventLog)
	return res
}

func (r *RPCReporter) buidUpStreamFromInputChannels() <-chan *olympuspb.ZoneUpStream {
	res := make(chan *olympuspb.ZoneUpStream, 10)

	mayPush := func(m *olympuspb.ZoneUpStream) {
		select {
		case res <- m:
		default:
		}
	}

	go func() {
		defer close(res)
		for {
			select {
			case target, ok := <-r.climateTargets:
				if ok == false {
					r.climateTargets = nil
				} else {
					mayPush(&olympuspb.ZoneUpStream{
						Target: buildOlympusClimateTarget(target),
					})
				}
			case report, ok := <-r.climateReports:
				if ok == false {
					r.climateReports = nil
				} else {
					mayPush(&olympuspb.ZoneUpStream{
						Reports: []*olympuspb.ClimateReport{buildOlympusClimateReport(report)}})
				}
			case event, ok := <-r.alarmReports:
				if ok == false {
					r.alarmReports = nil
				} else {
					mayPush(&olympuspb.ZoneUpStream{
						Alarms: []*olympuspb.AlarmEvent{buildOlympusAlarmEvent(event)},
					})
				}
			}
			if r.climateReports == nil && r.climateTargets == nil && r.alarmReports == nil {
				return
			}
		}
	}()
	return res
}

func (r *RPCReporter) Report(ready chan<- struct{}) {
	c, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &olympuspb.ZoneConnection{}
	defer func() {
		// need to capture the variable, not the method call
		conn.CloseAll(r.log)
	}()

	dialOptions := []grpc.DialOption{
		grpc.WithConnectParams(
			grpc.ConnectParams{
				MinConnectTimeout: 20 * time.Second,
				Backoff: backoff.Config{
					BaseDelay:  500 * time.Millisecond,
					Multiplier: backoff.DefaultConfig.Multiplier,
					Jitter:     backoff.DefaultConfig.Jitter,
					MaxDelay:   2 * time.Second,
				},
			},
		),
	}

	connections, connErrors := olympuspb.ConnectZoneAsync(nil, r.addr, r.declaration, r.log, dialOptions...)

	upstream := r.buidUpStreamFromInputChannels()
	var backlogs <-chan *olympuspb.ZoneUpStream
	var err error

	close(ready)
	for {
		if conn.Established() == false && connections == nil && connErrors == nil {
			time.Sleep(time.Duration((2.0 + 0.2*rand.Float64()) * float64(time.Second)))
			r.log.Printf("gRPC reconnection")
			connections, connErrors = olympuspb.ConnectZoneAsync(conn.ClienConn(),
				r.addr,
				r.declaration,
				r.log,
				dialOptions...)
		}

		select {
		case m, ok := <-upstream:
			if ok == false {
				return
			} else {
				if len(m.Reports) == 1 {
					r.lastReport = m.Reports[0]
				}
				if m.Target != nil {
					r.lastTarget = m.Target
				}
				_, err = conn.Send(m)
			}
		case m, ok := <-backlogs:
			if ok == false {
				backlogs = nil
			} else {
				_, err = conn.Send(m)
			}
		case co, ok := <-connections:
			if ok == false {
				connections = nil
			} else {
				r.log.Printf("gRPC connected")
				conn = co
				if r.connected != nil {
					r.connected <- true
				}
				if r.lastTarget != nil {
					_, err = conn.Send(&olympuspb.ZoneUpStream{Target: r.lastTarget})
				}
				backlogs = r.paginateBacklogs(c, conn.Confirmation().RegistrationConfirmation)
			}
		case connErr, ok := <-connErrors:
			if ok == false {
				connErrors = nil
			} else {
				r.log.Printf("gRPC connection failure: %s", connErr)
				conn.CloseAll(r.log)
			}
		}

		if err != nil {
			r.log.Printf("gRPC failure: %s", err)
			err = nil
			conn.CloseStream(r.log)
		}
	}
}

type RPCReporterOptions struct {
	zone           string
	olympusAddress string
	climate        zeus.ZoneClimate
	host           string
	runner         ZoneClimateRunner
}

func (o *RPCReporterOptions) sanitize(hostname string) {
	if len(o.host) == 0 {
		o.host = hostname
	}
}

func NewRPCReporter(o RPCReporterOptions) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	o.sanitize(hostname)

	logger := log.New(os.Stderr, "[zone/"+o.zone+"/rpc] ", 0)

	declaration := &olympuspb.ZoneDeclaration{
		Host:           o.host,
		Name:           o.zone,
		MinTemperature: zeus.AsFloat32Pointer(o.climate.MinimalTemperature),
		MaxTemperature: zeus.AsFloat32Pointer(o.climate.MaximalTemperature),
		MinHumidity:    zeus.AsFloat32Pointer(o.climate.MinimalHumidity),
		MaxHumidity:    zeus.AsFloat32Pointer(o.climate.MaximalHumidity),
	}
	return &RPCReporter{
		addr:           o.olympusAddress,
		declaration:    declaration,
		climateReports: make(chan zeus.ClimateReport, 20),
		alarmReports:   make(chan zeus.AlarmEvent, 20),
		climateTargets: make(chan zeus.ClimateTarget, 20),
		log:            logger,
		runner:         o.runner,
	}, nil
}
