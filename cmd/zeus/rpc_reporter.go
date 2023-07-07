package main

import (
	"context"
	"os"
	"path"
	"time"

	olympuspb "github.com/formicidae-tracker/olympus/pkg/api"
	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPCReporter struct {
	declaration *olympuspb.ClimateDeclaration
	addr        string
	lastReport  *olympuspb.ClimateReport
	lastTarget  *olympuspb.ClimateTarget

	runner ZoneClimateRunner

	climateReports chan zeus.ClimateReport
	alarmReports   chan zeus.AlarmEvent
	climateTargets chan zeus.ClimateTarget
	connected      chan bool

	log *logrus.Entry
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

func buildBackLog(reports []zeus.ClimateReport, events []zeus.AlarmEvent) *olympuspb.ClimateUpStream {
	res := &olympuspb.ClimateUpStream{
		Reports: make([]*olympuspb.ClimateReport, len(reports)),
		Alarms:  make([]*olympuspb.AlarmUpdate, len(events)),
		Backlog: true,
	}
	for i, r := range reports {
		res.Reports[i] = buildOlympusClimateReport(r)
	}
	for i, e := range events {
		res.Alarms[i] = buildOlympusAlarmUpdate(e)
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

func buildOlympusAlarmUpdate(event zeus.AlarmEvent) *olympuspb.AlarmUpdate {
	status := olympuspb.AlarmStatus_ON
	if event.Status == zeus.AlarmOff {
		status = olympuspb.AlarmStatus_OFF
	}
	level := olympuspb.AlarmLevel_WARNING
	if event.Flags&zeus.Emergency != 0x00 {
		level = olympuspb.AlarmLevel_EMERGENCY
	}
	return &olympuspb.AlarmUpdate{
		Identification: event.Identifier,
		Description:    event.Description,
		Status:         status,
		Time:           timestamppb.New(event.Time),
		Level:          level,
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
	ch chan<- *olympuspb.ClimateUpStream,
	pageSize int,
	reports []zeus.ClimateReport,
	events []zeus.AlarmEvent) {

	defer close(ch)

	push := func(m *olympuspb.ClimateUpStream) bool {
		select {
		case <-c.Done():
			return true
		case ch <- m:
			time.Sleep(50 * time.Millisecond)
		}
		return false
	}

	if pageSize <= 0 {
		push(buildBackLog(reports, events))
		return
	}

	for i := 0; i < len(reports); i += pageSize {
		end := i + pageSize
		if end > len(reports) {
			end = len(reports)
		}
		if push(buildBackLog(reports[i:end], nil)) == true {
			return
		}
	}
	for i := 0; i < len(events); i += pageSize {
		end := i + pageSize
		if end > len(events) {
			end = len(events)
		}

		if push(buildBackLog(nil, events[i:end])) == true {
			return
		}

	}
}

func (r *RPCReporter) paginateBacklogs(c context.Context,
	confirmation *olympuspb.ClimateRegistrationConfirmation) <-chan *olympuspb.ClimateUpStream {
	if confirmation == nil || r.runner == nil {
		return nil
	}

	if confirmation.PageSize <= 0 && confirmation.SendBacklogs == false {
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
	res := make(chan *olympuspb.ClimateUpStream)
	go paginateAsync(c, res, int(confirmation.PageSize), climateLog, eventLog)
	return res
}

func (r *RPCReporter) buidUpStreamFromInputChannels() <-chan *olympuspb.ClimateUpStream {
	res := make(chan *olympuspb.ClimateUpStream, 10)

	mayPush := func(m *olympuspb.ClimateUpStream) {
		select {
		case res <- m:
		default:
		}
	}

	go func() {
		defer close(res)
		for {

			if r.climateReports == nil && r.climateTargets == nil && r.alarmReports == nil {
				return
			}

			select {
			case target, ok := <-r.climateTargets:
				if ok == false {
					r.climateTargets = nil
				} else {
					mayPush(&olympuspb.ClimateUpStream{
						Target: buildOlympusClimateTarget(target),
					})
				}
			case report, ok := <-r.climateReports:
				if ok == false {
					r.climateReports = nil
				} else {
					mayPush(&olympuspb.ClimateUpStream{
						Reports: []*olympuspb.ClimateReport{buildOlympusClimateReport(report)}})
				}
			case event, ok := <-r.alarmReports:
				if ok == false {
					r.alarmReports = nil
				} else {
					mayPush(&olympuspb.ClimateUpStream{
						Alarms: []*olympuspb.AlarmUpdate{buildOlympusAlarmUpdate(event)},
					})
				}
			}
		}
	}()
	return res
}

func (r *RPCReporter) Report(ready chan<- struct{}) {
	ctx, cancelTask := context.WithCancel(context.Background())
	defer cancelTask()

	var cancelBacklog context.CancelFunc = func() {}

	upstream := r.buidUpStreamFromInputChannels()

	var backlogs <-chan *olympuspb.ClimateUpStream
	task := olympuspb.NewClimateTask(ctx, r.addr, r.declaration)

	go func() {
		if err := task.Run(); err != nil {
			r.log.WithError(err).Error("gRPC task error")
		}
	}()

	pushAndLogError := func(m *olympuspb.ClimateUpStream) {
		var res olympuspb.RequestResult[*olympuspb.ClimateDownStream]
		if len(m.Alarms) > 0 && m.Backlog == false {
			res = <-task.Request(m)
		} else {
			res = <-task.MayRequest(m)
		}
		if res.Error != nil {
			r.log.WithError(res.Error).Error("fort.olympus.Olympus/ClimateUpStream error")
		}
	}

	close(ready)

	for {
		select {
		case up, ok := <-upstream:
			if ok == false {
				cancelBacklog()
				return
			}
			go pushAndLogError(up)
		case up, ok := <-backlogs:
			if ok == false {
				backlogs = nil
			}
			go pushAndLogError(up)
		case down, ok := <-task.Confirmations():
			if ok == false {
				cancelBacklog()
				return
			}
			if down.Error == nil {
				r.log.Info("connected")
				cancelBacklog()
				var backlogContext context.Context
				backlogContext, cancelBacklog = context.WithCancel(ctx)
				backlogs = r.paginateBacklogs(backlogContext,
					down.Confirmation.RegistrationConfirmation)
			} else {
				r.log.WithError(down.Error).Warn("connection failure")
			}

			// for unit test purpose only
			if r.connected != nil {
				r.connected <- down.Error == nil
			}
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

	logger := tm.NewLogger(path.Join("zone", o.zone, "rpc"))

	declaration := &olympuspb.ClimateDeclaration{
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
