package main

import (
	"log"
	"os"
	"path"
	"sync"
	"time"

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
	log   *log.Logger
}

func buildRegisteredAlarm(ae *dieu.AlarmEvent) RegisteredAlarm {
	res := RegisteredAlarm{
		Reason:     ae.Reason,
		Level:      dieu.MapPriority(ae.Priority),
		LastChange: &time.Time{},
		Triggers:   0,
		On:         false,
	}
	*res.LastChange = ae.Time
	return res
}

func (z *ZoneData) registerAlarm(ae *dieu.AlarmEvent) {
	if _, ok := z.alarmMap[ae.Reason]; ok == true {
		return
	}

	z.alarmMap[ae.Reason] = len(z.zone.Alarms)

	z.zone.Alarms = append(z.zone.Alarms, buildRegisteredAlarm(ae))
}

func (h *Hermes) RegisterZone(reg *dieu.ZoneRegistration, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if z, ok := h.zones[reg.Fullname()]; ok == true {
		//close everything
		close(z.climate.Inbound())
		delete(h.zones, reg.Fullname())
	}
	h.log.Printf("Registering %s", reg.Fullname())
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
	go func() {
		res.climate.Sample()
	}()

	h.zones[reg.Fullname()] = res

	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ZoneIsRegistered(reg *dieu.ZoneUnregistration, ok *bool) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, *ok = h.zones[reg.Fullname()]
	return nil
}

func (h *Hermes) UnregisterZone(reg *dieu.ZoneUnregistration, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[reg.Fullname()]
	if ok == false {
		*err = dieu.HermesError(ZoneNotFoundError(reg.Fullname()).Error())
		return nil
	}
	h.log.Printf("Unregistering  %s", reg.Fullname())
	//it will close Sample go routine
	close(z.climate.Inbound())
	delete(h.zones, reg.Fullname())

	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportClimate(cr *dieu.NamedClimateReport, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[cr.ZoneIdentifier]
	if ok == false {
		*err = dieu.HermesError(ZoneNotFoundError(cr.ZoneIdentifier).Error())
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

func (h *Hermes) ReportAlarm(ae *dieu.AlarmEvent, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[ae.Zone]
	if ok == false {
		*err = dieu.HermesError(ZoneNotFoundError(ae.Zone).Error())
		return nil
	}
	aIdx, ok := z.alarmMap[ae.Reason]
	if ok == false {
		z.registerAlarm(ae)
		aIdx = z.alarmMap[ae.Reason]
	}

	h.log.Printf("New alarm event %+v", ae)
	if ae.Status == dieu.AlarmOn {
		if z.zone.Alarms[aIdx].On == false {
			z.zone.Alarms[aIdx].Triggers += 1
		}
		z.zone.Alarms[aIdx].On = true
	} else {
		z.zone.Alarms[aIdx].On = false
	}
	z.zone.Alarms[aIdx].LastChange = &time.Time{}
	*z.zone.Alarms[aIdx].LastChange = ae.Time
	//TODO: notify

	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportState(sr *dieu.StateReport, err *dieu.HermesError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	z, ok := h.zones[sr.Zone]
	if ok == false {
		*err = dieu.HermesError(ZoneNotFoundError(sr.Zone).Error())
		return nil
	}

	h.log.Printf("New state %+v", sr)
	if z.zone.Current == nil {
		z.zone.Current = &dieu.State{}
	}
	*z.zone.Current = sr.Current
	z.zone.CurrentEnd = z.zone.CurrentEnd
	if sr.Next != nil && sr.NextTime != nil {
		z.zone.Next = sr.Next
		z.zone.NextTime = sr.NextTime
		z.zone.NextEnd = sr.NextEnd
	} else {
		z.zone.Next = nil
		z.zone.NextEnd = nil
		z.zone.NextTime = nil
	}

	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) getZones() []RegisteredZone {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	res := make([]RegisteredZone, 0, len(h.zones))
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
		log:   log.New(os.Stderr, "[rpc] :", log.LstdFlags),
	}
}
