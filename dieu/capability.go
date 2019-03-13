package main

import (
	"fmt"
	"time"

	"git.tuleu.science/fort/dieu"
	"git.tuleu.science/fort/libarke/src-go/arke"
)

type StampedMessage struct {
	M  arke.ReceivableMessage
	T  time.Time
	ID arke.NodeID
}

type callback func(c chan<- dieu.Alarm, m *StampedMessage) error

type capability interface {
	Requirements() []arke.NodeClass
	SetDevices(devices map[arke.NodeClass]*Device)
	Action(s dieu.State) error
	Callbacks() map[arke.MessageClass]callback
	Alarms() []dieu.Alarm
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

func (c *ClimateControllable) Alarms() []dieu.Alarm {
	res := []dieu.Alarm{
		dieu.SensorReadoutIssue,
		dieu.TemperatureUnreachable,
		dieu.NewFanAlarm(zeusFanNames[0], arke.FanStalled),
		dieu.NewFanAlarm(zeusFanNames[1], arke.FanStalled),
		dieu.NewFanAlarm(zeusFanNames[2], arke.FanStalled),
	}
	if c.withCelaeno == true {
		res = append(res, dieu.WaterLevelUnreadable)
		res = append(res, dieu.WaterLevelCritical)
		res = append(res, dieu.WaterLevelWarning)
		res = append(res, dieu.HumidityUnreachable)
		res = append(res, dieu.NewFanAlarm("Celaeno Fan", arke.FanStalled))
	}
	return res
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

var zeusFanNames = []string{"Zeus Wind", "Zeus Extrcation Left", "Zeus Extraction Right"}

func (c ClimateControllable) Action(s dieu.State) error {
	if c.withCelaeno == true {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(dieu.Clamp(s.Temperature)),
			Humidity:    float32(dieu.Clamp(s.Humidity)),
			Wind:        uint8(dieu.Clamp(s.Wind) / 100.0 * 255),
		}
	} else {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(dieu.Clamp(s.Temperature)),
			Humidity:    float32(s.Humidity.MinValue()),
			Wind:        uint8(dieu.Clamp(s.Wind) / 100.0 * 255),
		}
	}
	return c.zeus.SendMessage(c.lastSetPoint)
}

func (c *ClimateControllable) Callbacks() map[arke.MessageClass]callback {
	res := map[arke.MessageClass]callback{}
	if c.withCelaeno == true {
		res[arke.CelaenoStatusMessage] = func(alarms chan<- dieu.Alarm, mm *StampedMessage) error {
			m, ok := mm.M.(*arke.CelaenoStatus)
			if ok == false {
				return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
			}
			if m.WaterLevel != arke.CelaenoWaterNominal {
				if m.WaterLevel&arke.CelaenoWaterReadError != 0 {
					alarms <- dieu.WaterLevelUnreadable
				} else if m.WaterLevel&arke.CelaenoWaterCritical != 0 {
					alarms <- dieu.WaterLevelCritical
				} else {
					alarms <- dieu.WaterLevelWarning
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
					alarms <- dieu.NewFanAlarm("Celaeno Fan", m.Fan.Status())
				}
			}
			return nil
		}
	}
	res[arke.ZeusStatusMessage] = func(alarms chan<- dieu.Alarm, mm *StampedMessage) error {
		m, ok := mm.M.(*arke.ZeusStatus)
		if ok == false {
			return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
		}

		if m.Status&arke.ZeusClimateNotControlledWatchDog != 0 {
			if m.Status&arke.ZeusActive != 0 {
				alarms <- dieu.SensorReadoutIssue
			} else if c.lastSetPoint != nil {
				if err := c.zeus.SendMessage(c.lastSetPoint); err != nil {
					return err
				}
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
				alarms <- dieu.HumidityUnreachable
			}
		}

		for i, f := range m.Fans {
			if f.Status() != arke.FanOK {
				if time.Now().After(c.zeusResetGuard) {
					c.zeusResetGuard = time.Now().Add(FanResetWindow)
					if err := c.zeus.SendResetRequest(); err != nil {
						return err
					}
					//give time for the device to reset
					time.Sleep(100 * time.Millisecond)
					if err := c.zeus.SendHeartbeatRequest(); err != nil {
						return err
					}

					if c.lastSetPoint != nil {
						return c.zeus.SendMessage(c.lastSetPoint)
					}
				} else {
					alarms <- dieu.NewFanAlarm(zeusFanNames[i], f.Status())
				}
			}
		}

		if m.Status&(arke.ZeusTemperatureUnreachable) != 0 {
			alarms <- dieu.TemperatureUnreachable
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

func (c *LightControllable) Alarms() []dieu.Alarm {
	return []dieu.Alarm{}
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

func (c *LightControllable) Action(s dieu.State) error {
	return c.helios.SendMessage(&arke.HeliosSetPoint{
		Visible: uint8(dieu.Clamp(s.VisibleLight) * 255 / 100),
		UV:      uint8(dieu.Clamp(s.UVLight) * 255 / 100),
	})
}

func (c *LightControllable) Callbacks() map[arke.MessageClass]callback {
	return nil
}

type ClimateRecordable struct {
	MinTemperature dieu.Temperature
	MaxTemperature dieu.Temperature
	MinHumidity    dieu.Humidity
	MaxHumidity    dieu.Humidity
	Notifiers      []chan<- dieu.ClimateReport
}

func NewClimateRecordableCapability(minT, maxT dieu.Temperature, minH, maxH dieu.Humidity, notifiers []chan<- dieu.ClimateReport) capability {
	res := &ClimateRecordable{
		MinTemperature: minT,
		MaxTemperature: maxT,
		MinHumidity:    minH,
		MaxHumidity:    maxH,
		Notifiers:      notifiers,
	}

	return res
}

func (r *ClimateRecordable) Requirements() []arke.NodeClass {
	return []arke.NodeClass{arke.ZeusClass}
}

func (c *ClimateRecordable) Alarms() []dieu.Alarm {
	return []dieu.Alarm{dieu.TemperatureOutOfBound, dieu.HumidityOutOfBound}
}

func (r *ClimateRecordable) SetDevices(map[arke.NodeClass]*Device) {}

func (r *ClimateRecordable) Action(s dieu.State) error { return nil }

func checkBound(v, min, max dieu.BoundedUnit) bool {
	if dieu.IsUndefined(min) == false && v.Value() < min.Value() {
		return false
	}

	if dieu.IsUndefined(max) == false && v.Value() > max.Value() {
		return false
	}

	return true
}

func (r *ClimateRecordable) Callbacks() map[arke.MessageClass]callback {
	return map[arke.MessageClass]callback{
		arke.ZeusReportMessage: func(alarms chan<- dieu.Alarm, mm *StampedMessage) error {
			report, ok := mm.M.(*arke.ZeusReport)
			if ok == false {
				return fmt.Errorf("Invalid Message Type %v", mm.M.MessageClassID())
			}

			if checkBound(dieu.Humidity(report.Humidity), r.MinHumidity, r.MaxHumidity) == false {
				alarms <- dieu.HumidityOutOfBound
			}

			if checkBound(dieu.Temperature(report.Temperature[0]), r.MinTemperature, r.MaxTemperature) == false {
				alarms <- dieu.TemperatureOutOfBound
			}

			for _, n := range r.Notifiers {
				n <- dieu.ClimateReport{
					Time:     mm.T,
					Humidity: dieu.Humidity(report.Humidity),
					Temperatures: [4]dieu.Temperature{
						dieu.Temperature(report.Temperature[0]),
						dieu.Temperature(report.Temperature[1]),
						dieu.Temperature(report.Temperature[2]),
						dieu.Temperature(report.Temperature[3]),
					},
				}
			}

			return nil
		},
	}

}
