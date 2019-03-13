package dieu

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
	RepeatPeriod() time.Duration
}

type AlarmString struct {
	p      Priority
	reason string
}

func (a AlarmString) Priority() Priority {
	return a.p
}

func (a AlarmString) Reason() string {
	return a.reason
}

func (a AlarmString) RepeatPeriod() time.Duration {
	return 500 * time.Millisecond
}

var WaterLevelWarning = AlarmString{Warning, "Celaeno water level is low"}
var WaterLevelCritical = AlarmString{Emergency, "Celaeno water level is empty"}
var WaterLevelUnreadable = AlarmString{Emergency, "Celaeno water level is unreadable"}
var HumidityUnreachable = AlarmString{Warning, "Cannot reach desired humidity"}
var TemperatureUnreachable = AlarmString{Warning, "Cannot reach desired humidity"}
var HumidityOutOfBound = AlarmString{Emergency, "Humidity is outside of boundaries"}
var TemperatureOutOfBound = AlarmString{Emergency, "Temperature is outside of boundaries"}
var SensorReadoutIssue = AlarmString{Emergency, "Sensors cannot be read"}

type MissingDeviceAlarm struct {
	canInterface string
	class        arke.NodeClass
	id           arke.NodeID
}

func (a MissingDeviceAlarm) Priority() Priority {
	return Emergency
}

func (a MissingDeviceAlarm) Reason() string {
	return fmt.Sprintf("Device '%s', with ID %d is missing on bus '%s'", arke.ClassName(a.class), a.id, a.canInterface)
}

func (a MissingDeviceAlarm) RepeatPeriod() time.Duration {
	return HeartBeatPeriod
}

func (a MissingDeviceAlarm) Device() (string, arke.NodeClass, arke.NodeID) {
	return a.canInterface, a.class, a.id
}

func NewMissingDeviceAlarm(intf string, c arke.NodeClass, id arke.NodeID) MissingDeviceAlarm {
	return MissingDeviceAlarm{intf, c, id}
}

type FanAlarm struct {
	fan    string
	status arke.FanStatus
}

func (a FanAlarm) Priority() Priority {
	if a.status == arke.FanStalled {
		return Emergency
	}
	return Warning
}

func (a FanAlarm) Reason() string {
	status := "aging"
	if a.status == arke.FanStalled {
		status = "stalled"
	}

	return fmt.Sprintf("Fan %s is %s", a.fan, status)
}

func (a FanAlarm) RepeatPeriod() time.Duration {
	return 500 * time.Millisecond
}

func (a FanAlarm) Fan() string {
	return a.fan
}

func (a FanAlarm) Status() arke.FanStatus {
	return a.status
}

func NewFanAlarm(fan string, s arke.FanStatus) FanAlarm {
	return FanAlarm{fan, s}
}

type AlarmStatus int

const (
	AlarmOn AlarmStatus = iota
	AlarmOff
)

type AlarmEvent struct {
	Zone   string
	Alarm  Alarm
	Status AlarmStatus
	Time   time.Time
}
