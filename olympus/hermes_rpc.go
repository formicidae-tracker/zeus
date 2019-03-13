package main

import (
	"fmt"
	"log"
	"path"
	"sync"

	"git.tuleu.science/fort/dieu"
)

type ZoneNotFoundError string

func NewZoneNotFoundError(fullname string) ZoneNotFoundError {
	return ZoneNotFoundError("hermes: unknwon zone '" + fullname + "'")
}

func (z ZoneNotFoundError) Error() string {
	return string(z)
}

type ZoneData struct {
	zone RegisteredZone

	climate  ClimateReportManager
	alarmMap map[string]int
}

type Hermes struct {
	mutex *sync.RWMutex
	zones map[string]*ZoneData
}

func (h *Hermes) RegisterZone(reg *dieu.ZoneRegistration, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if z, ok := h.zones[reg.Fullname()]; ok == true {
		//close everything
		close(z.climate.Inbound())
		delete(h.zones, reg.Fullname())
	}
	log.Printf("[rpc] Registering %s", reg.Fullname())
	res := &ZoneData{
		zone: RegisteredZone{
			Host:        reg.Host,
			Name:        reg.Name,
			Temperature: 0.0,
			TemperatureBounds: Bounds{
				nil, nil,
			},
			Humidity: 0.0,
			HumidityBounds: Bounds{
				nil, nil,
			},
		},
		climate:  NewClimateReportManager(),
		alarmMap: make(map[string]int),
	}
	for _, a := range reg.Alarms {
		regAlarm := RegisteredAlarm{
			Reason:     a.Reason,
			On:         false,
			LastChange: nil,
			Triggers:   0,
		}
		if a.Priority == dieu.Warning {
			regAlarm.Level = 1
		} else {
			regAlarm.Level = 2
		}
		res.alarmMap[a.Reason] = len(res.zone.Alarms)
		res.zone.Alarms = append(res.zone.Alarms, regAlarm)
	}
	go func() {
		res.climate.Sample()
	}()

	h.zones[reg.Fullname()] = res

	return nil
}

func (h *Hermes) UnregisterZone(reg *dieu.ZoneUnregistration, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[reg.Fullname()]
	if ok == false {
		dieu.ReturnError(err, ZoneNotFoundError(reg.Fullname()))
		return nil
	}
	log.Printf("[rpc] Unregistering  %s", reg.Fullname())
	//it will close Sample go routine
	close(z.climate.Inbound())
	delete(h.zones, reg.Fullname())

	dieu.ReturnError(err, nil)
	return nil
}

func (h *Hermes) ReportClimate(cr *dieu.NamedClimateReport, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[cr.ZoneIdentifier]
	if ok == false {
		dieu.ReturnError(err, ZoneNotFoundError(cr.ZoneIdentifier))
		return nil
	}

	z.zone.Temperature = float64((*cr).Temperatures[0])
	z.zone.Humidity = float64((*cr).Humidity)

	z.climate.Inbound() <- dieu.ClimateReport{
		Time:         cr.Time,
		Humidity:     cr.Humidity,
		Temperatures: [4]dieu.Temperature{cr.Temperatures[0], cr.Temperatures[1], cr.Temperatures[2], cr.Temperatures[3]},
	}
	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportAlarm(ae *dieu.HermesAlarmEvent, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[ae.ZoneIdentifier]
	if ok == false {
		dieu.ReturnError(err, ZoneNotFoundError(ae.ZoneIdentifier))
		return nil
	}
	aIdx, ok := z.alarmMap[ae.Reason]
	if ok == false {
		dieu.ReturnError(err, fmt.Errorf("hermes: Zone '%s' does not defines alarm '%s'", ae.ZoneIdentifier, ae.Reason))
		return nil
	}

	log.Printf("[rpc] New alarm event %v", ae)
	if ae.Status == true {
		if z.zone.Alarms[aIdx].On == false {
			z.zone.Alarms[aIdx].Triggers += 1
		}
		z.zone.Alarms[aIdx].On = true
	} else {
		z.zone.Alarms[aIdx].On = false
	}
	*z.zone.Alarms[aIdx].LastChange = ae.Time

	//TODO: notify

	dieu.ReturnError(err, nil)
	return nil
}

func (h *Hermes) getZones() []RegisteredZone {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	res := make([]RegisteredZone, 0, len(h.zones))
	log.Printf("I have %d zones", len(h.zones))
	for _, z := range h.zones {
		toAppend := RegisteredZone{
			Host: z.zone.Host,
			Name: z.zone.Name,
		}
		res = append(res, toAppend)
	}

	return res
}

func (h *Hermes) getZone(host, name string) (*RegisteredZone, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	fname := path.Join(host, "zone", name)
	z, ok := h.zones[fname]
	if ok == false {
		return nil, NewZoneNotFoundError(fname)
	}
	res := &RegisteredZone{}
	*res = z.zone
	return res, nil
}

func (h *Hermes) getClimateReport(host, name, window string) (ClimateReportTimeSerie, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	fname := path.Join(host, "zone", name)
	z, ok := h.zones[fname]
	if ok == false {
		return ClimateReportTimeSerie{}, NewZoneNotFoundError(fname)
	}

	switch window {
	case "hour":
		return z.climate.LastHour(), nil
	case "day":
		return z.climate.LastDay(), nil
	case "week":
		return z.climate.LastWeek(), nil
	default:
		return z.climate.LastHour(), nil
	}

}

func NewHermes() *Hermes {
	return &Hermes{
		mutex: &sync.RWMutex{},
		zones: make(map[string]*ZoneData),
	}
}
