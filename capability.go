package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
)

type StampedMessage struct {
	M  arke.ReceivableMessage
	D  time.Duration
	ID arke.NodeID
}

type callback func(c chan<- Alarm, m *StampedMessage) error

type capability interface {
	Requirements() []arke.NodeClass
	SetDevices(devices map[arke.NodeClass]*Device)
	Action(s State) error
	Callbacks() map[arke.MessageClass]callback
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

var zeusFanNames = []string{"Zeus Extraction Right", "Zeus Wind", "Zeus Extraction Left"}

func (c ClimateControllable) Action(s State) error {
	if c.withCelaeno == true {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(Clamp(s.Temperature)),
			Humidity:    float32(Clamp(s.Humidity)),
			Wind:        uint8(Clamp(s.Wind) / 100.0 * 255),
		}
	} else {
		c.lastSetPoint = &arke.ZeusSetPoint{
			Temperature: float32(Clamp(s.Temperature)),
			Humidity:    float32(s.Humidity.MinValue()),
			Wind:        uint8(Clamp(s.Wind) / 100.0 * 255),
		}
	}
	return c.zeus.SendMessage(c.lastSetPoint)
}

func (c *ClimateControllable) Callbacks() map[arke.MessageClass]callback {
	res := map[arke.MessageClass]callback{}
	if c.withCelaeno == true {
		res[arke.CelaenoStatusMessage] = func(alarms chan<- Alarm, mm *StampedMessage) error {
			m, ok := mm.M.(*arke.CelaenoStatus)
			if ok == false {
				return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
			}
			if m.WaterLevel != arke.CelaenoWaterNominal {
				if m.WaterLevel&arke.CelaenoWaterReadError != 0 {
					alarms <- WaterLevelUnreadable
				} else if m.WaterLevel&arke.CelaenoWaterCritical != 0 {
					alarms <- WaterLevelCritical
				} else {
					alarms <- WaterLevelWarning
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
					alarms <- NewFanAlarm("Celaeno Fan", m.Fan.Status())
				}
			}
			return nil
		}

	}
	res[arke.ZeusStatusMessage] = func(alarms chan<- Alarm, mm *StampedMessage) error {
		m, ok := mm.M.(*arke.ZeusStatus)
		if ok == false {
			return fmt.Errorf("Invalid message type %v", mm.M.MessageClassID())
		}

		if m.Status&arke.ZeusClimateNotControlledWatchDog != 0 {
			if c.lastSetPoint != nil {
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
				alarms <- HumidityUnreachable
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
					alarms <- NewFanAlarm(zeusFanNames[i], f.Status())
				}
			}
		}

		if m.Status&(arke.ZeusTemperatureUnreachable) != 0 {
			alarms <- TemperatureUnreachable
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

func (c *LightControllable) Action(s State) error {
	return c.helios.SendMessage(&arke.HeliosSetPoint{
		Visible: uint8(Clamp(s.VisibleLight) * 255 / 100),
		UV:      uint8(Clamp(s.UVLight) * 255 / 100),
	})
}

func (c *LightControllable) Callbacks() map[arke.MessageClass]callback {
	return nil
}

type ClimateRecordable struct {
	MinTemperature Temperature
	MaxTemperature Temperature
	MinHumidity    Humidity
	MaxHumidity    Humidity
	File           *os.File
}

func NewClimateRecordableCapability(minT, maxT Temperature, minH, maxH Humidity, file string) (capability, error) {
	res := &ClimateRecordable{
		MinTemperature: minT,
		MaxTemperature: maxT,
		MinHumidity:    minH,
		MaxHumidity:    maxH,
	}

	if len(file) > 0 {
		var err error
		var fname string
		res.File, fname, err = CreateFileWithoutOverwrite(file)
		if err != nil {
			return nil, err
		}
		log.Printf("Will save climate data in '%s'", fname)
		fmt.Fprintf(res.File, "#Starting date %s\n#Time(ms) Relative Humidity (%%) Temperature (°C) Temperature (°C) Temperature (°C) Temperature (°C)\n", time.Now())
	}
	return res, nil
}

func (r *ClimateRecordable) Requirements() []arke.NodeClass {
	return []arke.NodeClass{arke.ZeusClass}
}

func (r *ClimateRecordable) SetDevices(map[arke.NodeClass]*Device) {}

func (r *ClimateRecordable) Action(s State) error { return nil }

func checkBound(v, min, max BoundedUnit) bool {
	if IsUndefined(min) == false && v.Value() < min.Value() {
		return false
	}

	if IsUndefined(max) == false && v.Value() > max.Value() {
		return false
	}

	return true
}

func (r *ClimateRecordable) Callbacks() map[arke.MessageClass]callback {
	return map[arke.MessageClass]callback{
		arke.ZeusReportMessage: func(alarms chan<- Alarm, mm *StampedMessage) error {
			report, ok := mm.M.(*arke.ZeusReport)
			if ok == false {
				return fmt.Errorf("Invalid Message Type %v", mm.M.MessageClassID())
			}

			if r.File != nil {
				fmt.Fprintf(r.File,
					"%d %.2f %.2f %.2f %.2f %.2f\n",
					mm.D.Nanoseconds()/1e6,
					report.Humidity,
					report.Temperature[0],
					report.Temperature[1],
					report.Temperature[2],
					report.Temperature[3])
			}

			if checkBound(Humidity(report.Humidity), r.MinHumidity, r.MaxHumidity) == false {
				alarms <- HumidityOutOfBound
			}

			if checkBound(Temperature(report.Temperature[0]), r.MinTemperature, r.MaxTemperature) == false {
				alarms <- TemperatureOutOfBound
			}

			return nil
		},
	}

}
