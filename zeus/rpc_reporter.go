package main

import (
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type RPCReporter struct {
	Registration       zeus.ZoneRegistration
	Addr               string
	Conn               *rpc.Client
	LastStateReport    *zeus.StateReport
	ClimateReports     chan zeus.ClimateReport
	AlarmReports       chan zeus.AlarmEvent
	StateReports       chan zeus.StateReport
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

func (r *RPCReporter) StateChannel() chan<- zeus.StateReport {
	return r.StateReports
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

func (r *RPCReporter) Report(wg *sync.WaitGroup) {
	defer wg.Done()
	var rerr error
	trials := 0
	var resetConnection <-chan time.Time = nil
	var resetTimer *time.Timer = nil
	unused := 0
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
				ncr := zeus.NamedClimateReport{cr, r.Registration.Fullname()}
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
				if rerr == nil && trials <= r.MaxAttempts && resetConnection == nil {
					rerr = r.Conn.Call("Olympus.ReportAlarm", ae, &unused)
					if rerr != nil {
						r.log.Printf("Could not transmit alarm event: %s", rerr)
					}
				}
			}
		case sr, ok := <-r.StateReports:
			if ok == false {
				r.StateReports = nil
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
		if r.AlarmReports == nil && r.ClimateReports == nil && r.StateReports == nil {
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

func NewRPCReporter(name, address string, zone zeus.Zone, logs io.Writer) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(logs, "[zone/"+name+"/rpc] ", 0)

	logger.Printf("Opening connection to '%s'", address)

	conn, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("rpc: conn: %s", err)
	}

	unused := 0
	reg := zeus.ZoneRegistration{
		Host: hostname,
		Name: name,
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

	reg.Host = hostname
	reg.Name = name
	reg.MinHumidity = cast(zone.MinimalHumidity)
	reg.MaxHumidity = cast(zone.MaximalHumidity)
	reg.MinTemperature = cast(zone.MinimalTemperature)
	reg.MaxTemperature = cast(zone.MaximalTemperature)

	rerr := conn.Call("Olympus.RegisterZone", reg, &unused)
	if rerr != nil {
		return nil, fmt.Errorf("rpc: Olympus.RegisterZone: %s", rerr)
	}

	return &RPCReporter{
		Registration:       reg,
		Conn:               conn,
		Addr:               address,
		ClimateReports:     make(chan zeus.ClimateReport, 20),
		AlarmReports:       make(chan zeus.AlarmEvent, 20),
		StateReports:       make(chan zeus.StateReport, 20),
		log:                logger,
		ReconnectionWindow: 5 * time.Second,
		MaxAttempts:        1000,
	}, nil
}
