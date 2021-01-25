package main

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
)

type callback func(c chan<- TimedAlarm, m *StampedMessage) error

type capability interface {
	Requirements() []arke.NodeClass
	SetDevices(devices map[arke.NodeClass]*Device)
	Action(s zeus.State) error
	Callbacks() map[arke.MessageClass]callback
	Close() error
}

type ClimateControllable struct {
	withCelaeno       bool
	lastSetPoint      *arke.ZeusSetPoint
	celaeno           *Device
	zeus              *Device
	celaenoResetGuard time.Time
	zeusResetGuard    time.Time
}

func NewClimateControllable(forceHumidity bool) *ClimateControllable {
	return &ClimateControllable{
		celaenoResetGuard: time.Now(),
		zeusResetGuard:    time.Now(),
		withCelaeno:       forceHumidity,
	}
}

func (c *ClimateControllable) Requirements() []arke.NodeClass {
	if c.withCelaeno == true {
		return []arke.NodeClass{arke.ZeusClass, arke.CelaenoClass}
	}
	return []arke.NodeClass{arke.ZeusClass}
}

func (c *ClimateControllable) SetDevices(devices map[arke.NodeClass]*Device) {
	c.zeus = devices[arke.ZeusClass]
	if c.withCelaeno {
		c.celaeno = devices[arke.CelaenoClass]
		if c.celaeno == nil {
			panic("Celaeno is missing")
		}

	}
	if c.zeus == nil {
		panic("Zeus is missing")
	}
}

func (c *ClimateControllable) Close() error {
	return nil
}

var zeusFanNames = []string{"Zeus Wind", "Zeus Extraction Right", "Zeus Extraction Left"}

func (c *ClimateControllable) Action(s zeus.State) error {
	if c.withCelaeno == true {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(zeus.Clamp(s.Temperature)),
			Humidity:    float32(zeus.Clamp(s.Humidity)),
			Wind:        uint8(zeus.Clamp(s.Wind) / 100.0 * 255),
		}
	} else {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(zeus.Clamp(s.Temperature)),
			Humidity:    float32(s.Humidity.MinValue()),
			Wind:        uint8(zeus.Clamp(s.Wind) / 100.0 * 255),
		}
	}
	return c.zeus.SendMessage(c.lastSetPoint)
}

func (c *ClimateControllable) Callbacks() map[arke.MessageClass]callback {
	res := map[arke.MessageClass]callback{}
	if c.withCelaeno == true {
		res[arke.CelaenoStatusMessage] = func(alarms chan<- TimedAlarm, mm *StampedMessage) error {
			m, ok := mm.M.(*arke.CelaenoStatus)
			if ok == false {
				return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
			}
			if m.WaterLevel != arke.CelaenoWaterNominal {
				if m.WaterLevel&arke.CelaenoWaterReadError != 0 {
					alarms <- TimedAlarm{Alarm: zeus.WaterLevelUnreadable, Time: mm.T}
				} else if m.WaterLevel&arke.CelaenoWaterCritical != 0 {
					alarms <- TimedAlarm{Alarm: zeus.WaterLevelCritical, Time: mm.T}
				} else {
					alarms <- TimedAlarm{Alarm: zeus.WaterLevelWarning, Time: mm.T}
				}
			}
			if m.Fan.Status() != arke.FanOK {
				if time.Now().After(c.celaenoResetGuard) {
					c.celaenoResetGuard = time.Now().Add(FanResetWindow)
					if err := c.celaeno.SendResetRequest(); err != nil {
						return err
					}
					time.Sleep(100 * time.Millisecond)

					return c.celaeno.SendHeartbeatRequest()
				} else {
					alarms <- TimedAlarm{Alarm: zeus.NewFanAlarm("Celaeno Fan", m.Fan.Status()), Time: mm.T}
				}
			}
			return nil
		}
	}
	res[arke.ZeusStatusMessage] = func(alarms chan<- TimedAlarm, mm *StampedMessage) error {
		m, ok := mm.M.(*arke.ZeusStatus)
		if ok == false {
			return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
		}

		if m.Status&arke.ZeusClimateNotControlledWatchDog != 0 {
			if m.Status&arke.ZeusActive != 0 {
				alarms <- TimedAlarm{Alarm: zeus.SensorReadoutIssue, Time: mm.T}
				if time.Now().After(c.zeusResetGuard) {
					c.zeusResetGuard = time.Now().Add(FanResetWindow)
					c.zeus.SendResetRequest()
				}
			} else if c.lastSetPoint != nil {
				if err := c.zeus.SendMessage(c.lastSetPoint); err != nil {
					return err
				}
			} else {
				alarms <- TimedAlarm{Alarm: zeus.ClimateStateUndefined, Time: mm.T}
			}
		}

		if m.Status&arke.ZeusHumidityUnreachable != 0 {
			if time.Now().After(c.celaenoResetGuard) {
				c.celaenoResetGuard = time.Now().Add(FanResetWindow)
				if err := c.celaeno.SendResetRequest(); err != nil {
					return err
				}
				time.Sleep(100 * time.Millisecond)

				return c.celaeno.SendHeartbeatRequest()
			} else {
				alarms <- TimedAlarm{Alarm: zeus.HumidityUnreachable, Time: mm.T}
			}
		}

		for i, f := range m.Fans {
			if f.Status() != arke.FanOK {
				alarms <- TimedAlarm{Alarm: zeus.NewFanAlarm(zeusFanNames[i], f.Status()), Time: mm.T}
			}
		}

		if m.Status&(arke.ZeusTemperatureUnreachable) != 0 {
			alarms <- TimedAlarm{Alarm: zeus.TemperatureUnreachable, Time: mm.T}
		}
		return nil
	}
	return res
}

type LightControllable struct {
	helios *Device
}

func NewLightControllable() *LightControllable {
	return &LightControllable{}
}

func (c *LightControllable) Requirements() []arke.NodeClass {
	return []arke.NodeClass{arke.HeliosClass}
}

func (c *LightControllable) SetDevices(devices map[arke.NodeClass]*Device) {
	c.helios = devices[arke.HeliosClass]
	if c.helios == nil {
		panic("Missing Helios Module")
	}
}

func (c *LightControllable) Action(s zeus.State) error {
	return c.helios.SendMessage(&arke.HeliosSetPoint{
		Visible: uint8(zeus.Clamp(s.VisibleLight) * 255 / 100),
		UV:      uint8(zeus.Clamp(s.UVLight) * 255 / 100),
	})
}

func (c *LightControllable) Close() error {
	return nil
}

func (c *LightControllable) Callbacks() map[arke.MessageClass]callback {
	return nil
}

type ClimateRecordable struct {
	MinTemperature zeus.Temperature
	MaxTemperature zeus.Temperature
	MinHumidity    zeus.Humidity
	MaxHumidity    zeus.Humidity
	Notifiers      []chan<- zeus.ClimateReport
}

func NewClimateRecordableCapability(minT, maxT zeus.Temperature, minH, maxH zeus.Humidity, notifiers []chan<- zeus.ClimateReport) capability {
	res := &ClimateRecordable{
		MinTemperature: minT,
		MaxTemperature: maxT,
		MinHumidity:    minH,
		MaxHumidity:    maxH,
		Notifiers:      notifiers,
	}

	return res
}

func (r *ClimateRecordable) Close() error {
	for _, n := range r.Notifiers {
		close(n)
	}
	return nil
}

func (r *ClimateRecordable) Requirements() []arke.NodeClass {
	return []arke.NodeClass{arke.ZeusClass}
}

func (r *ClimateRecordable) SetDevices(map[arke.NodeClass]*Device) {}

func (r *ClimateRecordable) Action(s zeus.State) error { return nil }

func checkBound(v, min, max zeus.BoundedUnit) bool {
	if zeus.IsUndefined(min) == false && v.Value() < min.Value() {
		return false
	}

	if zeus.IsUndefined(max) == false && v.Value() > max.Value() {
		return false
	}

	return true
}

func (r *ClimateRecordable) Callbacks() map[arke.MessageClass]callback {
	return map[arke.MessageClass]callback{
		arke.ZeusReportMessage: func(alarms chan<- TimedAlarm, mm *StampedMessage) error {
			report, ok := mm.M.(*arke.ZeusReport)
			if ok == false {
				return fmt.Errorf("Invalid Message Type %v", mm.M.MessageClassID())
			}

			if checkBound(zeus.Humidity(report.Humidity), r.MinHumidity, r.MaxHumidity) == false {
				alarms <- TimedAlarm{Alarm: zeus.HumidityOutOfBound, Time: mm.T}
			}

			if checkBound(zeus.Temperature(report.Temperature[0]), r.MinTemperature, r.MaxTemperature) == false {
				alarms <- TimedAlarm{Alarm: zeus.TemperatureOutOfBound, Time: mm.T}
			}

			creport := zeus.ClimateReport{
				Time:     mm.T,
				Humidity: zeus.Humidity(report.Humidity),
				Temperatures: [4]zeus.Temperature{
					zeus.Temperature(report.Temperature[0]),
					zeus.Temperature(report.Temperature[1]),
					zeus.Temperature(report.Temperature[2]),
					zeus.Temperature(report.Temperature[3]),
				},
			}

			if creport.Good() == true {
				for _, n := range r.Notifiers {
					n <- creport
				}
			}

			return nil
		},
	}

}

func ComputeClimateRequirements(z *zeus.ZoneClimate, reporters []ClimateReporter) []capability {
	res := []capability{}

	needClimateReport := len(reporters) > 0
	if zeus.IsUndefined(z.MinimalTemperature) == false || zeus.IsUndefined(z.MaximalTemperature) == false {
		needClimateReport = true
	}
	if zeus.IsUndefined(z.MinimalHumidity) == false || zeus.IsUndefined(z.MaximalHumidity) == false {
		needClimateReport = true
	}

	if needClimateReport == true {
		chans := []chan<- zeus.ClimateReport{}
		for _, n := range reporters {
			chans = append(chans, n.ReportChannel())
		}

		res = append(res, NewClimateRecordableCapability(z.MinimalTemperature,
			z.MaximalTemperature,
			z.MinimalHumidity,
			z.MaximalHumidity,
			chans))
	}

	controlLight := false
	controlTemperature := false
	controlHumidity := false
	controlWind := false

	for _, s := range z.States {
		if zeus.IsUndefined(s.Humidity) == false {
			controlHumidity = true
		}
		if zeus.IsUndefined(s.Temperature) == false {
			controlTemperature = true
		}
		if zeus.IsUndefined(s.Wind) == false {
			controlWind = true
		}
		if zeus.IsUndefined(s.VisibleLight) == false || zeus.IsUndefined(s.UVLight) == false {
			controlLight = true
		}
	}

	if controlTemperature == true || controlWind == true {
		res = append(res, NewClimateControllable(controlHumidity))
	}

	if controlLight == true {
		res = append(res, NewLightControllable())
	}

	return res
}
