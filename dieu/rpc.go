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
}

func (r *RPCReporter) ReportChannel() chan<- dieu.ClimateReport {
	return r.ClimateReports
}

func (r *RPCReporter) AlarmChannel() chan<- dieu.AlarmEvent {
	return r.AlarmReports
}

func (r *RPCReporter) reconnect() error {
	log.Printf("[%s] Reconnecting '%s'", r.Registration.Fullname(), r.Addr)
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
						log.Printf("[%s]: Could not transmit climate report: %s", r.Registration.Fullname(), rerr)
					}
					if herr.ToError() != nil {
						log.Printf("[%s]: Could not transmit climate report: %s", r.Registration.Fullname(), herr.ToError())
					}
				} else {
					log.Printf("[%s] Discarded Report Climate error: %s", r.Registration.Fullname(), rerr)
				}
			}
		case ae, ok := <-r.AlarmReports:
			if ok == false {
				r.AlarmReports = nil
			} else {
				if rerr == nil {
					rerr := r.Conn.Call("Hermes.ReportAlarm", ae, &herr)
					if rerr != nil {
						log.Printf("[%s]: Could not transmit alarm event: %s", r.Registration.Fullname(), rerr)
					}
					if herr.ToError() != nil {
						log.Printf("[%s]: Could not transmit alarm event: %s", r.Registration.Fullname(), herr.ToError())
					}
				} else {
					log.Printf("[%s] Discarded Alarm Event: error: %s", r.Registration.Fullname(), rerr)
				}
			}
		}
		if r.AlarmReports == nil && r.ClimateReports == nil {
			break
		}

		if rerr != nil {
			log.Printf("[%s] Disconnecting '%s' due to rpc error %s", r.Registration.Fullname(), r.Addr, rerr)
			r.Conn.Close()
		}
	}

	herr := dieu.HermesError("")
	rerr = r.Conn.Call("Hermes.UnregisterZone", &dieu.ZoneUnregistration{
		Name: r.Registration.Name,
		Host: r.Registration.Host,
	}, &herr)
	if rerr != nil {
		log.Printf("[%s]: Could not unregister zone: %s", r.Registration.Name, rerr)
	}
	if herr.ToError() != nil {
		log.Printf("[%s]: Could not unregister zone: %s", r.Registration.Name, herr.ToError())
	}
	r.Conn.Close()
}

func NewRPCReporter(name, address string, zone dieu.Zone) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

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
	}, nil
}
