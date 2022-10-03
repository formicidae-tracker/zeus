package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/formicidae-tracker/olympus/proto"
	"github.com/formicidae-tracker/zeus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPCReporter struct {
	Declaration     *proto.ZoneDeclaration
	Addr            string
	LastStateReport *zeus.ClimateTarget
	ClimateReports  chan zeus.ClimateReport
	AlarmReports    chan zeus.AlarmEvent
	ClimateTargets  chan zeus.ClimateTarget
	log             *log.Logger
	BackoffDelay    time.Duration
}

func sanitizeBoundedUnit(u zeus.BoundedUnit) *float32 {
	v := u.Value()
	if zeus.IsUndefined(u) || math.IsNaN(v) || math.IsInf(v, 1) || v <= -1000.0 {
		return nil
	}
	res := new(float32)
	*res = float32(v)
	return res
}

func (r *RPCReporter) ReportChannel() chan<- zeus.ClimateReport {
	return r.ClimateReports
}

func (r *RPCReporter) AlarmChannel() chan<- zeus.AlarmEvent {
	return r.AlarmReports
}

func (r *RPCReporter) TargetChannel() chan<- zeus.ClimateTarget {
	return r.ClimateTargets
}

func (r *RPCReporter) reconnect(conn *grpc.ClientConn) (*grpc.ClientConn, proto.Olympus_ZoneClient, error) {
	var err error
	if conn == nil {
		dialOptions := append(proto.DefaultDialOptions,
			grpc.WithConnectParams(
				grpc.ConnectParams{
					MinConnectTimeout: 5 * time.Second,
					Backoff: backoff.Config{
						BaseDelay:  100 * time.Millisecond,
						Multiplier: backoff.DefaultConfig.Multiplier,
						Jitter:     backoff.DefaultConfig.Jitter,
						MaxDelay:   1 * time.Second,
					},
				},
			))
		conn, err = grpc.Dial(r.Addr, dialOptions...)

		if err != nil {
			return nil, nil, err
		}
	}

	client := proto.NewOlympusClient(conn)
	stream, err := client.Zone(context.Background(), proto.DefaultCallOptions...)
	if err != nil {
		return conn, nil, err
	}

	err = send(stream, &proto.ZoneUpStream{
		Declaration: r.Declaration,
	})
	if err != nil {
		stream.CloseSend()
		return conn, nil, err
	}

	return conn, stream, nil
}

func buildOlympusClimateReport(report zeus.ClimateReport) *proto.ClimateReport {
	temperatures := make([]float32, len(report.Temperatures))
	for i, t := range report.Temperatures {
		temperatures[i] = float32(t)
	}
	return &proto.ClimateReport{
		Time:         timestamppb.New(report.Time),
		Humidity:     sanitizeBoundedUnit(report.Humidity),
		Temperatures: temperatures,
	}
}

func buildOlympusAlarmEvent(event zeus.AlarmEvent) *proto.AlarmEvent {
	status := proto.AlarmStatus_ALARM_ON
	if event.Status == zeus.AlarmOff {
		status = proto.AlarmStatus_ALARM_OFF
	}
	level := int32(1)
	if event.Flags&zeus.Emergency != 0x00 {
		level = int32(2)
	}
	return &proto.AlarmEvent{
		Reason: event.Reason,
		Status: status,
		Time:   timestamppb.New(event.Time),
		Level:  level,
	}
}

func buildOlympusState(s *zeus.State) *proto.ClimateState {
	if s == nil {
		return nil
	}
	return &proto.ClimateState{
		Temperature:  sanitizeBoundedUnit(s.Temperature),
		Humidity:     sanitizeBoundedUnit(s.Humidity),
		Wind:         sanitizeBoundedUnit(s.Wind),
		VisibleLight: sanitizeBoundedUnit(s.VisibleLight),
		UvLight:      sanitizeBoundedUnit(s.UVLight),
	}
}

func buildOlympusClimateTarget(target zeus.ClimateTarget) *proto.ClimateTarget {
	res := &proto.ClimateTarget{
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

func send(stream proto.Olympus_ZoneClient, m *proto.ZoneUpStream) error {
	if stream == nil {
		return nil
	}
	err := stream.Send(m)
	if err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}
	_, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("could not receive ack: %w", err)
	}
	return nil
}

func (r *RPCReporter) Report(ready chan<- struct{}) {
	trials := 0
	var resetConnection <-chan time.Time = nil
	var resetTimer *time.Timer = nil
	conn, stream, connErr := r.reconnect(nil)
	defer func() {
		if stream != nil {
			err := stream.CloseSend()
			if err != nil {
				r.log.Printf("could not send CloseSend(): %s", err)
			}
		}
		if conn != nil {
			err := conn.Close()
			if err != nil {
				r.log.Printf("could not close connection(): %s", err)
			}
		}
	}()

	var err error
	r.log.Printf("started")
	close(ready)
	for {
		if stream == nil {
			r.log.Printf("will reconnect in %s previous trials: %d.",
				r.BackoffDelay, trials)
			resetTimer = time.NewTimer(r.BackoffDelay)
			resetConnection = resetTimer.C
		}
		select {
		case <-resetConnection:
			trials += 1
			conn, stream, connErr = r.reconnect(conn)
			if connErr == nil {
				trials = 0
			} else {
				r.log.Printf("could not reconnect: %s", connErr)
			}
			resetTimer.Stop()
			resetConnection = nil
		case report, ok := <-r.ClimateReports:
			if ok == false {
				r.ClimateReports = nil
			} else {
				err = send(stream, &proto.ZoneUpStream{
					Reports: []*proto.ClimateReport{buildOlympusClimateReport(report)},
				})
			}
		case event, ok := <-r.AlarmReports:
			if ok == false {
				r.AlarmReports = nil
			} else {
				err = send(stream, &proto.ZoneUpStream{
					Alarms: []*proto.AlarmEvent{buildOlympusAlarmEvent(event)},
				})
			}
		case target, ok := <-r.ClimateTargets:
			if ok == false {
				r.ClimateTargets = nil
			} else {
				r.LastStateReport = &target
				err = send(stream, &proto.ZoneUpStream{
					Target: buildOlympusClimateTarget(target),
				})
			}
		}
		if r.AlarmReports == nil && r.ClimateReports == nil && r.ClimateTargets == nil {
			break
		}

		if err != nil {
			r.log.Printf("gRPC failure: %s", err)
			err = stream.CloseSend()
			if err != nil {
				r.log.Printf("gRPC CloseSend() failure: %s", err)
			}
			r.log.Printf("re-building stream after gRPC")
			conn, stream, connErr = r.reconnect(conn)
			if connErr != nil {
				r.log.Printf("closing connection after gRPC request failure: %s", connErr)
				if conn != nil {
					connErr = conn.Close()
					if connErr != nil {
						r.log.Printf("could not close connection: %s", connErr)
					}
					conn = nil
				}
			}
		}
	}
}

type RPCReporterOptions struct {
	zone           string
	olympusAddress string
	climate        zeus.ZoneClimate
	host           string
	rpcPort        int
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

	declaration := &proto.ZoneDeclaration{
		Host:           o.host,
		Name:           o.zone,
		MinTemperature: sanitizeBoundedUnit(o.climate.MinimalTemperature),
		MaxTemperature: sanitizeBoundedUnit(o.climate.MaximalTemperature),
		MinHumidity:    sanitizeBoundedUnit(o.climate.MinimalHumidity),
		MaxHumidity:    sanitizeBoundedUnit(o.climate.MaximalHumidity),
	}

	return &RPCReporter{
		Addr:           fmt.Sprintf("%s:%d", o.olympusAddress, o.rpcPort),
		Declaration:    declaration,
		ClimateReports: make(chan zeus.ClimateReport, 20),
		AlarmReports:   make(chan zeus.AlarmEvent, 20),
		ClimateTargets: make(chan zeus.ClimateTarget, 20),
		log:            logger,
		BackoffDelay:   5 * time.Second,
	}, nil
}
