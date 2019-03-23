package main

import (
	"time"

	"github.com/formicidae-tracker/dieu"
	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type RegisteredAlarm struct {
	Reason     string
	On         bool
	Level      int
	LastChange *time.Time
	Triggers   int
}

type Bounds struct {
	Min *float64
	Max *float64
}

type RegisteredZone struct {
	Host              string
	Name              string
	Temperature       float64
	TemperatureBounds Bounds
	Humidity          float64
	HumidityBounds    Bounds
	Alarms            []RegisteredAlarm
	Current           *dieu.State
	CurrentEnd        *dieu.State
	Next              *dieu.State
	NextEnd           *dieu.State
	NextTime          *time.Time
}

var stubZone RegisteredZone

func init() {
	stubZone = RegisteredZone{
		Temperature: 21.2,
		TemperatureBounds: Bounds{
			Min: new(float64),
			Max: new(float64),
		},
		Humidity: 62.0,
		HumidityBounds: Bounds{
			Min: new(float64),
			Max: new(float64),
		},
		Alarms: []RegisteredAlarm{},
	}
	*stubZone.TemperatureBounds.Min = 22.0
	*stubZone.TemperatureBounds.Max = 28.0
	*stubZone.HumidityBounds.Min = 40.0
	*stubZone.HumidityBounds.Max = 75.0

	alarms := []dieu.Alarm{
		dieu.WaterLevelWarning,
		dieu.WaterLevelCritical,
		dieu.TemperatureOutOfBound,
		dieu.HumidityOutOfBound,
		dieu.TemperatureUnreachable,
		dieu.HumidityUnreachable,
		dieu.NewMissingDeviceAlarm("slcan0", arke.ZeusClass, 1),
		dieu.NewMissingDeviceAlarm("slcan0", arke.CelaenoClass, 1),
		dieu.NewMissingDeviceAlarm("slcan0", arke.HeliosClass, 1),
	}
	for _, a := range alarms {
		aa := RegisteredAlarm{
			Reason:   a.Reason(),
			On:       false,
			Triggers: 0,
		}
		if a.Priority() == dieu.Warning {
			aa.Level = 1
		} else {
			aa.Level = 2
		}

		stubZone.Alarms = append(stubZone.Alarms, aa)
	}

}
