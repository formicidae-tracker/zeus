package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"path"

	"git.tuleu.science/fort/dieu"
)

type RPCReporter struct {
	Name           string
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

func (r *RPCReporter) Report() {
	for {
		var herr dieu.HermesError
		select {
		case cr, ok := <-r.ClimateReports:
			if ok == false {
				r.ClimateReports = nil
			} else {
				ncr := dieu.NamedClimateReport{cr, r.Name}
				rerr := r.Conn.Call("Hermes.ReportClimate", ncr, &herr)
				if rerr != nil {
					log.Printf("[%s]: Could not transmit climate report: %s", r.Name, rerr)
				}
				if herr.ToError() != nil {
					log.Printf("[%s]: Could not transmit climate report: %s", r.Name, herr.ToError())
				}

			}
		case ae, ok := <-r.AlarmReports:
			if ok == false {
				r.AlarmReports = nil
			} else {
				toSend := dieu.HermesAlarmEvent{
					Time:           ae.Time,
					Reason:         ae.Alarm.Reason(),
					ZoneIdentifier: ae.Zone,
					Status:         false,
				}
				if ae.Status == dieu.AlarmOn {
					toSend.Status = true
				}
				rerr := r.Conn.Call("Hermes.ReportAlarm", toSend, &herr)
				if rerr != nil {
					log.Printf("[%s]: Could not transmit alarm event: %s", r.Name, rerr)
				}
				if herr.ToError() != nil {
					log.Printf("[%s]: Could not transmit alarm event: %s", r.Name, herr.ToError())
				}
			}
		}
		if r.AlarmReports == nil && r.ClimateReports == nil {
			break
		}
	}

	name := path.Base(r.Name)
	host := path.Dir(path.Dir(r.Name))
	herr := dieu.HermesError("")
	rerr := r.Conn.Call("Hermes.UnregisterZone", &dieu.ZoneUnregistration{
		Name: name,
		Host: host,
	}, &herr)
	if rerr != nil {
		log.Printf("[%s]: Could not unregister zone: %s", r.Name, rerr)
	}
	if herr.ToError() != nil {
		log.Printf("[%s]: Could not unregister zone: %s", r.Name, herr.ToError())
	}
	r.Conn.Close()
}

func NewRPCReporter(name, address string, alarms []dieu.Alarm) (*RPCReporter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	fullname := path.Join(hostname, "zone", name)

	conn, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("rpc: conn: %s", err)
	}

	herr := dieu.HermesError("")

	rerr := conn.Call("Hermes.RegisterZone", &dieu.ZoneRegistration{
		Host: hostname,
		Name: name,
	}, &herr)
	if rerr != nil {
		return nil, fmt.Errorf("rpc: call: %s", rerr)
	}
	if herr.ToError() != nil {
		return nil, fmt.Errorf("rpc: Hermes.RegisterZone: %s", herr.ToError())
	}

	return &RPCReporter{
		Name:           fullname,
		Conn:           conn,
		ClimateReports: make(chan dieu.ClimateReport, 20),
		AlarmReports:   make(chan dieu.AlarmEvent, 20),
	}, nil
}
