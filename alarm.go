package main

import (
	"fmt"
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
)

type Priority int

const (
	Emergency Priority = iota
	Warning
)

type Alarm interface {
	Priority() Priority
	Reason() string
	RepeatInterval() time.Duration
}

type alarmString struct {
	p      Priority
	reason string
}

func (a alarmString) Priority() Priority {
	return a.p
}

func (a alarmString) Reason() string {
	return a.reason
}

func (a alarmString) RepeatInterval() time.Duration {
	return 500 * time.Millisecond
}

var WaterLevelWarning = alarmString{Warning, "Celaeno water level is low"}
var WaterLevelCritical = alarmString{Emergency, "Celaeno water level is empty"}
var WaterLevelUnreadable = alarmString{Emergency, "Celaeno water level is unreadable"}
var HumidityUnreachable = alarmString{Warning, "Cannot reach desired humidity"}
var TemperatureUnreachable = alarmString{Warning, "Cannot reach desired humidity"}
var HumidityOutOfBound = alarmString{Emergency, "Humidity is outside of boundaries"}
var TemperatureOutOfBound = alarmString{Emergency, "Temperature is outside of boundaries"}

type MissingDeviceAlarm interface {
	Alarm
	Device() (string, arke.NodeClass, arke.NodeID)
}

type missingDeviceAlarm struct {
	canInterface string
	class        arke.NodeClass
	id           arke.NodeID
}

func (a missingDeviceAlarm) Priority() Priority {
	return Emergency
}

func (a missingDeviceAlarm) Reason() string {
	return fmt.Sprintf("Device '%s', with ID %d is missing on bus '%s'", Name(a.class), a.id, a.canInterface)
}

func (a missingDeviceAlarm) RepeatInterval() time.Duration {
	return HeartBeatPeriod
}

func (a missingDeviceAlarm) Device() (string, arke.NodeClass, arke.NodeID) {
	return a.canInterface, a.class, a.id
}

func NewMissingDeviceAlarm(intf string, c arke.NodeClass, id arke.NodeID) MissingDeviceAlarm {
	return missingDeviceAlarm{intf, c, id}
}

type FanAlarm interface {
	Alarm
	Fan() string
	Status() arke.FanStatus
}

type fanAlarm struct {
	fan    string
	status arke.FanStatus
}

func (a fanAlarm) Priority() Priority {
	if a.status == arke.FanStalled {
		return Emergency
	}
	return Warning
}

func (a fanAlarm) Reason() string {
	status := "aging"
	if a.status == arke.FanStalled {
		status = "stalled"
	}

	return fmt.Sprintf("Fan %s is %s", a.fan, status)
}

func (a fanAlarm) RepeatInterval() time.Duration {
	return 500 * time.Millisecond
}

func (a fanAlarm) Fan() string {
	return a.fan
}

func (a fanAlarm) Status() arke.FanStatus {
	return a.status
}

func NewFanAlarm(fan string, s arke.FanStatus) FanAlarm {
	return fanAlarm{fan, s}
}
