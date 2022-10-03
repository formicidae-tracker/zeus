package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type RPCReporter struct {
	Registration       zeus.ZoneRegistration
	Addr               string
	Conn               *rpc.Client
	LastStateReport    *zeus.ClimateTarget
	ClimateReports     chan zeus.ClimateReport
	AlarmReports       chan zeus.AlarmEvent
	ClimateTargets     chan zeus.ClimateTarget
	log                *log.Logger
	ReconnectionWindow time.Duration
	MaxAttempts        int
}

func (r *RPCReporter) ReportChannel() chan<- zeus.ClimateReport {
	return r.ClimateReports
}

func (r *RPCReporter) AlarmChannel() chan<- zeus.AlarmEvent {
	return r.AlarmReports
}

func (r *RPCReporter) StateChannel() chan<- zeus.ClimateTarget {
	return r.ClimateTargets
}

func (r *RPCReporter) reconnect() error {
	r.log.Printf("Reconnecting '%s'", r.Addr)
	var err error
	r.Conn, err = rpc.DialHTTP("tcp", r.Addr)
	if err != nil {
		return err
	}

	registered := false

	toSend := zeus.ZoneUnregistration{
		Host: r.Registration.Host,
		Name: r.Registration.Name,
	}
	err = r.Conn.Call("Olympus.ZoneIsRegistered", toSend, &registered)
	if err != nil {
		return err
	}

	if registered == true {
		return nil
	}
	unused := 0

	err = r.Conn.Call("Olympus.RegisterZone", r.Registration, &unused)
	if err != nil {
		return err
	}

	if r.LastStateReport == nil {
		return nil
	}

	return r.Conn.Call("Olympus.ReportState", r.LastStateReport, &unused)
}

func (r *RPCReporter) Report(ready chan<- struct{}) {
	var rerr error
	trials := 0
	var resetConnection <-chan time.Time = nil
	var resetTimer *time.Timer = nil
	unused := 0
	r.log.Printf("started")
	close(ready)
	for {
		if rerr != nil && resetConnection == nil {
			if trials < r.MaxAttempts {
				r.log.Printf("Will reconnect in %s previous trials: %d, max:%d", r.ReconnectionWindow, trials, r.MaxAttempts)
				resetTimer = time.NewTimer(r.ReconnectionWindow)
				resetConnection = resetTimer.C
			} else {
				log.Printf("Disabling connection after %d attemps", r.MaxAttempts)
				rerr = nil
			}
		}
		select {
		case <-resetConnection:
			trials += 1
			rerr = r.reconnect()
			if rerr == nil {
				trials = 0
			} else {
				r.log.Printf("Could not reconnect: %s", rerr)
			}
			resetTimer.Stop()
			resetConnection = nil
		case cr, ok := <-r.ClimateReports:
			if ok == false {
				r.ClimateReports = nil
			} else {
				r.Registration.SizeClimateLog++
				ncr := zeus.NamedClimateReport{cr, r.Registration.ZoneIdentifier()}
				if rerr == nil && trials <= r.MaxAttempts && resetConnection == nil {
					rerr = r.Conn.Call("Olympus.ReportClimate", ncr, &unused)
					if rerr != nil {
						r.log.Printf("Could not transmit climate report: %s", rerr)
					}
				}
			}
		case ae, ok := <-r.AlarmReports:
			if ok == false {
				r.AlarmReports = nil
			} else {
				r.Registration.SizeAlarmLog++
				if rerr == nil && trials <= r.MaxAttempts && resetConnection == nil {
					rerr = r.Conn.Call("Olympus.ReportAlarm", ae, &unused)
					if rerr != nil {
						r.log.Printf("Could not transmit alarm event: %s", rerr)
					}
				}
			}
		case sr, ok := <-r.ClimateTargets:
			if ok == false {
				r.ClimateTargets = nil
			} else {
				r.LastStateReport = &sr
				if rerr == nil && trials <= r.MaxAttempts && resetConnection == nil {
					rerr = r.Conn.Call("Olympus.ReportState", sr, &unused)
					if rerr != nil {
						r.log.Printf("Could not transmit state report: %s", rerr)
					}
				}
			}
		}
		if r.AlarmReports == nil && r.ClimateReports == nil && r.ClimateTargets == nil {
			break
		}

		if rerr != nil && resetConnection == nil && r.Conn != nil {
			r.log.Printf("Disconnecting '%s' due to rpc error %s", r.Addr, rerr)
			r.Conn.Close()
		}
	}

	if r.Conn == nil {
		//disconnected
		return
	}

	r.log.Printf("Unregistering zone")
	rerr = r.Conn.Call("Olympus.UnregisterZone", &zeus.ZoneUnregistration{
		Name: r.Registration.Name,
		Host: r.Registration.Host,
	}, &unused)
	if rerr != nil {
		r.log.Printf("Could not unregister zone: %s", rerr)
	}
	r.Conn.Close()
}

type RPCReporterOptions struct {
	zoneName       string
	olympusAddress string
	climate        zeus.ZoneClimate
	numAux         int
	wantedHostname string
	rpcPort        int
}

func (o *RPCReporterOptions) sanitize(hostname string) {
	if len(o.wantedHostname) == 0 {
		o.wantedHostname = hostname
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

	logger := log.New(os.Stderr, "[zone/"+o.zoneName+"/rpc] ", 0)

	logger.Printf("Opening connection to '%s'", o.olympusAddress)
	conn, err := rpc.DialHTTP("tcp", o.olympusAddress)
	if err != nil {
		return nil, fmt.Errorf("rpc: conn: %s", err)
	}

	unused := 0
	reg := zeus.ZoneRegistration{
		Host: o.wantedHostname,
		Name: o.zoneName,
	}

	cast := func(from zeus.BoundedUnit) *float64 {
		if zeus.IsUndefined(from) == true {
			return nil
		} else {
			res := new(float64)
			*res = from.Value()
			return res
		}
	}

	reg.Host = o.wantedHostname
	reg.Name = o.zoneName
	reg.MinHumidity = cast(o.climate.MinimalHumidity)
	reg.MaxHumidity = cast(o.climate.MaximalHumidity)
	reg.MinTemperature = cast(o.climate.MinimalTemperature)
	reg.MaxTemperature = cast(o.climate.MaximalTemperature)
	reg.NumAux = o.numAux
	reg.RPCAddress = fmt.Sprintf("%s.local:%d", hostname, o.rpcPort)

	rerr := conn.Call("Olympus.UnregisterZone", &zeus.ZoneUnregistration{
		Host: o.wantedHostname,
		Name: o.zoneName,
	}, &unused)

	if rerr != nil {
		logger.Printf("could not unregister zone: %s", rerr)
	}

	rerr = conn.Call("Olympus.RegisterZone", reg, &unused)
	if rerr != nil {
		return nil, fmt.Errorf("rpc: Olympus.RegisterZone: %s", rerr)
	}

	return &RPCReporter{
		Registration:       reg,
		Conn:               conn,
		Addr:               o.olympusAddress,
		ClimateReports:     make(chan zeus.ClimateReport, 20),
		AlarmReports:       make(chan zeus.AlarmEvent, 20),
		ClimateTargets:     make(chan zeus.ClimateTarget, 20),
		log:                logger,
		ReconnectionWindow: 5 * time.Second,
		MaxAttempts:        1000,
	}, nil
}
