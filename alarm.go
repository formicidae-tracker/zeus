package zeus

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type AlarmFlags int

const (
	Warning             AlarmFlags = 0x00
	Emergency                      = 0x01
	InstantNotification            = 0x80
)

type Alarm interface {
	Flags() AlarmFlags
	Reason() string
	DeadLine() time.Duration
}

type AlarmString struct {
	f        AlarmFlags
	reason   string
	deadline time.Duration
}

func (a AlarmString) Flags() AlarmFlags {
	return a.f
}

func (a AlarmString) Reason() string {
	return a.reason
}

func (a AlarmString) DeadLine() time.Duration {
	return a.deadline
}

var WaterLevelWarning = AlarmString{Warning | InstantNotification, "Celaeno water level is low", 2 * time.Second}
var WaterLevelCritical = AlarmString{Emergency | InstantNotification, "Celaeno is empty", 2 * time.Second}
var WaterLevelUnreadable = AlarmString{Emergency | InstantNotification, "Celaeno water level is unreadable", 2 * time.Second}
var HumidityUnreachable = AlarmString{Warning, "Cannot reach desired humidity", 10 * time.Minute}
var TemperatureUnreachable = AlarmString{Warning, "Cannot reach desired temperature", 10 * time.Minute}
var HumidityOutOfBound = AlarmString{Emergency | InstantNotification, "Humidity is outside of boundaries", 1 * time.Minute}
var TemperatureOutOfBound = AlarmString{Emergency | InstantNotification, "Temperature is outside of boundaries", 1 * time.Minute}
var SensorReadoutIssue = AlarmString{Emergency, "Cannot read sensors", 2 * time.Second}
var ClimateStateUndefined = AlarmString{Emergency, "Climate State Undefined", 2 * time.Second}

type MissingDeviceAlarm struct {
	canInterface string
	class        arke.NodeClass
	id           arke.NodeID
}

func (a MissingDeviceAlarm) Flags() AlarmFlags {
	return Emergency | InstantNotification
}

func (a MissingDeviceAlarm) Reason() string {
	return fmt.Sprintf("Device %s.%s.%d is missing", a.canInterface, arke.ClassName(a.class), a.id)
}

func (a MissingDeviceAlarm) DeadLine() time.Duration {
	return 5 * HeartBeatPeriod
}

func (a MissingDeviceAlarm) Device() (string, arke.NodeClass, arke.NodeID) {
	return a.canInterface, a.class, a.id
}

func NewMissingDeviceAlarm(intf string, c arke.NodeClass, id arke.NodeID) MissingDeviceAlarm {
	if id < 1 {
		id = 1
	} else if id > 7 {
		id = 7
	}
	return MissingDeviceAlarm{intf, c, id}
}

type FanAlarm struct {
	fan    string
	status arke.FanStatus
}

func (a FanAlarm) Flags() AlarmFlags {
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

func (a FanAlarm) DeadLine() time.Duration {
	return 10 * time.Minute
}

func (a FanAlarm) Fan() string {
	return a.fan
}

func (a FanAlarm) Status() arke.FanStatus {
	return a.status
}

func NewFanAlarm(fan string, s arke.FanStatus) FanAlarm {
	if s == arke.FanOK {
		s = arke.FanAging
	}
	return FanAlarm{fan, s}
}

type DeviceInternalError struct {
	intfName  string
	class     arke.NodeClass
	id        arke.NodeID
	errorCode uint16
}

func NewDeviceInternalError(intfName string, c arke.NodeClass, id arke.NodeID, e uint16) DeviceInternalError {
	return DeviceInternalError{intfName: intfName, class: c, id: id, errorCode: e}
}

func (e DeviceInternalError) Flags() AlarmFlags {
	return Warning
}

func (e DeviceInternalError) DeadLine() time.Duration {
	return 2 * time.Second
}

func (e DeviceInternalError) Reason() string {
	return fmt.Sprintf("Device %s.%s.%d internal error 0x%04x", e.intfName, e.class, e.id, e.errorCode)
}

func (e DeviceInternalError) Device() (string, arke.NodeClass, arke.NodeID) {
	return e.intfName, e.class, e.id
}

func (e DeviceInternalError) ErrorCode() uint16 {
	return e.errorCode
}

type AlarmStatus int

const (
	AlarmOn AlarmStatus = iota
	AlarmOff
)

type AlarmEvent struct {
	ZoneIdentifier string
	Reason         string
	Flags          AlarmFlags
	Status         AlarmStatus
	Time           time.Time
}

func MapPriority(f AlarmFlags) int {
	if f&Emergency != 0 {
		return 2
	}
	return 1
}
