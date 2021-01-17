package main

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
)

type StampedMessage struct {
	M  arke.ReceivableMessage
	T  time.Time
	ID arke.NodeID
}

type callback func(c chan<- zeus.Alarm, m *StampedMessage) error

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
		res[arke.CelaenoStatusMessage] = func(alarms chan<- zeus.Alarm, mm *StampedMessage) error {
			m, ok := mm.M.(*arke.CelaenoStatus)
			if ok == false {
				return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
			}
			if m.WaterLevel != arke.CelaenoWaterNominal {
				if m.WaterLevel&arke.CelaenoWaterReadError != 0 {
					alarms <- zeus.WaterLevelUnreadable
				} else if m.WaterLevel&arke.CelaenoWaterCritical != 0 {
					alarms <- zeus.WaterLevelCritical
				} else {
					alarms <- zeus.WaterLevelWarning
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
					alarms <- zeus.NewFanAlarm("Celaeno Fan", m.Fan.Status())
				}
			}
			return nil
		}
	}
	res[arke.ZeusStatusMessage] = func(alarms chan<- zeus.Alarm, mm *StampedMessage) error {
		m, ok := mm.M.(*arke.ZeusStatus)
		if ok == false {
			return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
		}

		if m.Status&arke.ZeusClimateNotControlledWatchDog != 0 {
			if m.Status&arke.ZeusActive != 0 {
				alarms <- zeus.SensorReadoutIssue
				if time.Now().After(c.zeusResetGuard) {
					c.zeusResetGuard = time.Now().Add(FanResetWindow)
					c.zeus.SendResetRequest()
				}
			} else if c.lastSetPoint != nil {
				if err := c.zeus.SendMessage(c.lastSetPoint); err != nil {
					return err
				}
			} else {
				alarms <- zeus.ClimateStateUndefined
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
				alarms <- zeus.HumidityUnreachable
			}
		}

		for i, f := range m.Fans {
			if f.Status() != arke.FanOK {
				alarms <- zeus.NewFanAlarm(zeusFanNames[i], f.Status())
			}
		}

		if m.Status&(arke.ZeusTemperatureUnreachable) != 0 {
			alarms <- zeus.TemperatureUnreachable
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
		arke.ZeusReportMessage: func(alarms chan<- zeus.Alarm, mm *StampedMessage) error {
			report, ok := mm.M.(*arke.ZeusReport)
			if ok == false {
				return fmt.Errorf("Invalid Message Type %v", mm.M.MessageClassID())
			}

			if checkBound(zeus.Humidity(report.Humidity), r.MinHumidity, r.MaxHumidity) == false {
				alarms <- zeus.HumidityOutOfBound
			}

			if checkBound(zeus.Temperature(report.Temperature[0]), r.MinTemperature, r.MaxTemperature) == false {
				alarms <- zeus.TemperatureOutOfBound
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
