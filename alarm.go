package dieu

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
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
var WaterLevelCritical = AlarmString{Emergency, "Celaeno is empty"}
var WaterLevelUnreadable = AlarmString{Emergency, "Celaeno water level is unreadable"}
var HumidityUnreachable = AlarmString{Warning, "Cannot reach desired humidity"}
var TemperatureUnreachable = AlarmString{Warning, "Cannot reach desired temperature"}
var HumidityOutOfBound = AlarmString{Emergency, "Humidity is outside of boundaries"}
var TemperatureOutOfBound = AlarmString{Emergency, "Temperature is outside of boundaries"}
var SensorReadoutIssue = AlarmString{Emergency, "Cannot read sensors"}
var ClimateStateUndefined = AlarmString{Emergency, "Climate State Undefined"}

type MissingDeviceAlarm struct {
	canInterface string
	class        arke.NodeClass
	id           arke.NodeID
}

func (a MissingDeviceAlarm) Priority() Priority {
	return Emergency
}

func (a MissingDeviceAlarm) Reason() string {
	return fmt.Sprintf("Device %s.%s.%d is missing", a.canInterface, arke.ClassName(a.class), a.id)
}

func (a MissingDeviceAlarm) RepeatPeriod() time.Duration {
	return HeartBeatPeriod
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

func (e DeviceInternalError) Priority() Priority {
	return Warning
}

func (e DeviceInternalError) RepeatPeriod() time.Duration {
	return 500 * time.Millisecond
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
	Zone     string
	Reason   string
	Priority Priority
	Status   AlarmStatus
	Time     time.Time
}

var levelByPriority map[Priority]int

func MapPriority(p Priority) int {
	if r, ok := levelByPriority[p]; ok == true {
		return r
	}
	return levelByPriority[Emergency]
}

func init() {
	levelByPriority = map[Priority]int{
		Warning:   1,
		Emergency: 2,
	}
}
