package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"git.tuleu.science/fort/dieu"
)

type RPCReporter struct {
	Registration   dieu.ZoneRegistration
	Addr           string
	Conn           *rpc.Client
	ClimateReports chan dieu.ClimateReport
	AlarmReports   chan dieu.AlarmEvent
	log            *log.Logger
}

func (r *RPCReporter) ReportChannel() chan<- dieu.ClimateReport {
	return r.ClimateReports
}

func (r *RPCReporter) AlarmChannel() chan<- dieu.AlarmEvent {
	return r.AlarmReports
}

func (r *RPCReporter) reconnect() error {
	r.log.Printf("Reconnecting '%s'", r.Addr)
	var err error
	r.Conn, err = rpc.DialHTTP("tcp", r.Addr)
	if err != nil {
		return err
	}

	registered := false

	toSend := dieu.ZoneUnregistration{
		Host: r.Registration.Host,
		Name: r.Registration.Name,
	}
	err = r.Conn.Call("Hermes.ZoneIsRegistered", toSend, &registered)
	if err != nil {
		return err
	}

	if registered == true {
		return nil
	}
	herr := dieu.HermesError("")

	err = r.Conn.Call("Hermes.RegisterZone", r.Registration, &herr)
	if err != nil {
		return err
	}

	return herr.ToError()
}

func (r *RPCReporter) Report() {
	var rerr error

	for {
		if rerr != nil {
			rerr = r.reconnect()
		}
		var herr dieu.HermesError
		select {
		case cr, ok := <-r.ClimateReports:
			if ok == false {
				r.ClimateReports = nil
			} else {
				ncr := dieu.NamedClimateReport{cr, r.Registration.Fullname()}
				if rerr == nil {
					rerr := r.Conn.Call("Hermes.ReportClimate", ncr, &herr)
					if rerr != nil {
						r.log.Printf("Could not transmit climate report: %s", rerr)
					}
					if herr.ToError() != nil {
						r.log.Printf("Could not transmit climate report: %s", herr.ToError())
					}
				} else {
					r.log.Printf("Discarded Report Climate error: %s", rerr)
				}
			}
		case ae, ok := <-r.AlarmReports:
			if ok == false {
				r.AlarmReports = nil
			} else {
				if rerr == nil {
					rerr := r.Conn.Call("Hermes.ReportAlarm", ae, &herr)
					if rerr != nil {
						r.log.Printf("Could not transmit alarm event: %s", rerr)
					}
					if herr.ToError() != nil {
						r.log.Printf("Could not transmit alarm event: %s", herr.ToError())
					}
				} else {
					r.log.Printf("Discarded Alarm Event: error: %s", rerr)
				}
			}
		}
		if r.AlarmReports == nil && r.ClimateReports == nil {
			break
		}

		if rerr != nil {
			r.log.Printf("Disconnecting '%s' due to rpc error %s", r.Addr, rerr)
			r.Conn.Close()
		}
	}

	herr := dieu.HermesError("")
	rerr = r.Conn.Call("Hermes.UnregisterZone", &dieu.ZoneUnregistration{
		Name: r.Registration.Name,
		Host: r.Registration.Host,
	}, &herr)
	if rerr != nil {
		r.log.Printf("Could not unregister zone: %s", rerr)
	}
	if herr.ToError() != nil {
		r.log.Printf("Could not unregister zone: %s", herr.ToError())
	}
	r.Conn.Close()
}

func NewRPCReporter(name, address string, zone dieu.Zone) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stderr, "[zone/"+name+"/rpc]:", log.LstdFlags)

	logger.Printf("Opening connection to '%s'", address)

	conn, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("rpc: conn: %s", err)
	}

	herr := dieu.HermesError("")
	reg := dieu.ZoneRegistration{
		Host: hostname,
		Name: name,
	}

	cast := func(from dieu.BoundedUnit) *float64 {
		if dieu.IsUndefined(from) == true {
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

	rerr := conn.Call("Hermes.RegisterZone", reg, &herr)
	if rerr != nil {
		return nil, fmt.Errorf("rpc: call: %s", rerr)
	}
	if herr.ToError() != nil {
		return nil, fmt.Errorf("rpc: Hermes.RegisterZone: %s", herr.ToError())
	}

	return &RPCReporter{
		Registration:   reg,
		Conn:           conn,
		Addr:           address,
		ClimateReports: make(chan dieu.ClimateReport, 20),
		AlarmReports:   make(chan dieu.AlarmEvent, 20),
		log:            logger,
	}, nil
}
