package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/barkimedes/go-deepcopy"
	olympuspb "github.com/formicidae-tracker/olympus/api"
	"github.com/formicidae-tracker/zeus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
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
	log            *log.Logger
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

type connectionData struct {
	conn   *grpc.ClientConn
	stream olympuspb.Olympus_ClimateClient
	err    error
}

func (d *connectionData) send(m *olympuspb.ClimateUpStream) error {
	if d.stream == nil {
		return nil
	}
	err := d.stream.Send(m)
	if err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}
	_, err = d.stream.Recv()
	if err != nil {
		return fmt.Errorf("could not receive ack: %w", err)
	}
	return nil
}

func (d *connectionData) closeAndLogErrors(logger *log.Logger) {
	if d.stream != nil {
		err := d.stream.CloseSend()
		if err != nil {
			logger.Printf("gRPC CloseSend() failure: %s", err)
		}
	}
	d.stream = nil
	if d.conn != nil {
		err := d.conn.Close()
		if err != nil {
			logger.Printf("gRPC close() failure: %s", err)
		}
	}
	d.conn = nil
}

func (r *RPCReporter) connect(conn *grpc.ClientConn,
	declaration *olympuspb.ClimateUpStream) (res connectionData) {
	defer func() {
		if res.err != nil {
			res.closeAndLogErrors(r.log)
		}
	}()

	if conn == nil {
		dialOptions := append(olympuspb.DefaultDialOptions,
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
			))
		res.conn, res.err = grpc.Dial(r.addr, dialOptions...)
		if res.err != nil {
			return
		}
	} else {
		// Ensure that we do not flood the server over the connection
		time.Sleep(500 * time.Millisecond)
	}

	client := olympuspb.NewOlympusClient(conn)
	res.stream, res.err = client.Climate(context.Background(), olympuspb.DefaultCallOptions...)
	if res.err != nil {
		return
	}

	res.err = res.stream.Send(declaration)
	if res.err != nil {
		return
	}

	var m *olympuspb.ClimateDownStream
	m, res.err = res.stream.Recv()
	if res.err != nil {
		return
	}

	res.err = r.sendBackLogs(&res, m.RegistrationConfirmation)
	return
}

func (r *RPCReporter) sendBackLogs(conn *connectionData, confirmation *olympuspb.ClimateRegistrationConfirmation) error {
	if conn.stream == nil || confirmation == nil || confirmation.SendBacklogs == false {
		return nil
	}
	if r.runner == nil {
		return fmt.Errorf("internal error: runner is not set")
	}

	climateLog, err := r.runner.ClimateLog(0, 0)
	if err != nil {
		return fmt.Errorf("could not get climate log: %w", err)
	}
	alarmLog, err := r.runner.AlarmLog(0, 0)
	if err != nil {
		return fmt.Errorf("could not get alarm log: %w", err)
	}

	if confirmation.PageSize == 0 {
		return conn.send(buildBackLog(climateLog, alarmLog))
	}

	for i := 0; i < len(climateLog); i += int(confirmation.PageSize) {
		end := i + int(confirmation.PageSize)
		if end > len(climateLog) {
			end = len(climateLog)
		}
		if err := conn.send(buildBackLog(climateLog[i:end], nil)); err != nil {
			return err
		}
	}

	for i := 0; i < len(alarmLog); i += int(confirmation.PageSize) {
		end := i + int(confirmation.PageSize)
		if end > len(alarmLog) {
			end = len(alarmLog)
		}
		if err := conn.send(buildBackLog(nil, alarmLog[i:end])); err != nil {
			return err
		}
	}

	return nil
}

func buildBackLog(reports []zeus.ClimateReport, events []zeus.AlarmEvent) *olympuspb.ClimateUpStream {
	res := &olympuspb.ClimateUpStream{
		Reports: make([]*olympuspb.ClimateReport, len(reports)),
		Alarms:  make([]*olympuspb.AlarmUpdate, len(events)),
	}
	for i, r := range reports {
		res.Reports[i] = buildOlympusClimateReport(r)
	}
	for i, e := range events {
		res.Alarms[i] = buildOlympusAlarmUpdate(e)
	}
	return res
}

func (r *RPCReporter) connectAsync(conn *grpc.ClientConn,
	declaration *olympuspb.ClimateUpStream) <-chan connectionData {

	res := make(chan connectionData)
	safeDeclaration := deepcopy.MustAnything(declaration).(*olympuspb.ClimateUpStream)
	go func() {
		res <- r.connect(conn, safeDeclaration)
		close(res)
	}()

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

func (r *RPCReporter) Report(ready chan<- struct{}) {
	connectionResult := r.connectAsync(nil, &olympuspb.ClimateUpStream{
		Declaration: r.declaration,
	})
	var conn connectionData
	defer func() {
		conn.closeAndLogErrors(r.log)
	}()

	var err error
	r.log.Printf("started")
	close(ready)
	for {
		if conn.stream == nil && connectionResult == nil {
			r.log.Printf("reconnecting gRPC")
			m := &olympuspb.ClimateUpStream{
				Declaration: r.declaration,
				Target:      r.lastTarget,
			}
			if r.lastReport != nil {
				m.Reports = []*olympuspb.ClimateReport{r.lastReport}
			}
			connectionResult = r.connectAsync(conn.conn, m)
		}
		select {
		case d, ok := <-connectionResult:
			if ok == false {
				connectionResult = nil
			} else {
				conn = d
				if conn.err != nil {
					r.log.Printf("gRPC could not connect: %s", err)
				}
			}
		case report, ok := <-r.climateReports:
			if ok == false {
				r.climateReports = nil
			} else {
				oReport := buildOlympusClimateReport(report)
				r.lastReport = oReport
				err = conn.send(&olympuspb.ClimateUpStream{
					Reports: []*olympuspb.ClimateReport{oReport},
				})
			}
		case event, ok := <-r.alarmReports:
			if ok == false {
				r.alarmReports = nil
			} else {
				err = conn.send(&olympuspb.ClimateUpStream{
					Alarms: []*olympuspb.AlarmUpdate{buildOlympusAlarmUpdate(event)},
				})
			}
		case target, ok := <-r.climateTargets:
			if ok == false {
				r.climateTargets = nil
			} else {
				oTarget := buildOlympusClimateTarget(target)
				r.lastTarget = oTarget
				err = conn.send(&olympuspb.ClimateUpStream{
					Target: oTarget,
				})
			}
		}

		if r.alarmReports == nil && r.climateReports == nil && r.climateTargets == nil {
			break
		}

		if err != nil {
			r.log.Printf("gRPC failure: %s", err)
			err = conn.stream.CloseSend()
			if err != nil {
				r.log.Printf("gRPC CloseSend() failure: %s", err)
			}
			conn.stream = nil
		}
	}
}

type RPCReporterOptions struct {
	zone           string
	olympusAddress string
	climate        zeus.ZoneClimate
	host           string
	rpcPort        int
	runner         ZoneClimateRunner
}

func (o *RPCReporterOptions) sanitize(hostname string) {
	if len(o.host) == 0 {
		o.host = hostname
	}
	if o.rpcPort <= 0 {
		o.rpcPort = zeus.ZEUS_PORT
	}
}

func NewRPCReporter(o RPCReporterOptions) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	o.sanitize(hostname)

	logger := log.New(os.Stderr, "[zone/"+o.zone+"/rpc] ", 0)

	declaration := &olympuspb.ClimateDeclaration{
		Host:           o.host,
		Name:           o.zone,
		MinTemperature: zeus.AsFloat32Pointer(o.climate.MinimalTemperature),
		MaxTemperature: zeus.AsFloat32Pointer(o.climate.MaximalTemperature),
		MinHumidity:    zeus.AsFloat32Pointer(o.climate.MinimalHumidity),
		MaxHumidity:    zeus.AsFloat32Pointer(o.climate.MaximalHumidity),
	}

	return &RPCReporter{
		addr:           fmt.Sprintf("%s:%d", o.olympusAddress, o.rpcPort),
		declaration:    declaration,
		climateReports: make(chan zeus.ClimateReport, 20),
		alarmReports:   make(chan zeus.AlarmEvent, 20),
		climateTargets: make(chan zeus.ClimateTarget, 20),
		log:            logger,
		runner:         o.runner,
	}, nil
}
