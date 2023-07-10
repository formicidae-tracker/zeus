package zeus

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
)

type AlarmFlags int

const (
	Warning   AlarmFlags = 0x00
	Emergency            = 0x01
	Failure              = 0x02
	AdminOnly            = 0x04
)

type Alarm interface {
	Flags() AlarmFlags
	Identifier() string
	Description() string
	MinUpTime() time.Duration
	MinDownTime() time.Duration
}

type AlarmString struct {
	f                      AlarmFlags
	identifier             string
	description            string
	minUpTime, minDownTime time.Duration
}

func (a AlarmString) Flags() AlarmFlags {
	return a.f
}

func (a AlarmString) Identifier() string {
	return a.identifier
}

func (a AlarmString) Description() string {
	return a.description
}

func (a AlarmString) MinUpTime() time.Duration {
	return a.minUpTime
}

func (a AlarmString) MinDownTime() time.Duration {
	return a.minDownTime
}

var WaterLevelWarning = AlarmString{Emergency, "climate.water_level", "Water tank level is low", 2 * time.Minute, 1 * time.Minute}
var WaterLevelCritical = AlarmString{Failure, "climate.water_level", "Water tank is empty", 2 * time.Minute, 1 * time.Minute}
var WaterLevelUnreadable = AlarmString{Failure, "climate.water_sensor", "Celaeno cannot determine water tank level", 2 * time.Minute, 1 * time.Minute}
var HumidityUnreachable = AlarmString{Warning, "climate.humidity.unreachable", "Cannot reach desired humidity", 20 * time.Minute, 10 * time.Minute}
var TemperatureUnreachable = AlarmString{Warning, "climate.temperature.unreachable", "Cannot reach desired temperature", 20 * time.Minute, 10 * time.Minute}

func OutOfBound[T Temperature | Humidity](min, max T) Alarm {
	identifier := ""
	name := ""
	unit := ""
	switch any(max).(type) {
	case Temperature:
		identifier = "climate.temperature.out_of_bounds"
		name = "Temperature"
		unit = "°C"
	case Humidity:
		identifier = "climate.humidity.out_of_bounds"
		name = "Humidity"
		unit = "% R.H."
	default:
		panic(fmt.Sprintf("unsupported type %T", max))
	}
	return AlarmString{
		f:          Emergency,
		identifier: identifier,
		description: fmt.Sprintf("%s is outside of boundaries ( [%.1f ; %.1f] %s )",
			name, min, max, unit),
		minUpTime:   2 * time.Minute,
		minDownTime: 1 * time.Minute,
	}
}

var SensorReadoutIssue = AlarmString{Failure, "climate.sensor.readout", "Cannot read sensors", 1 * time.Minute, 2 * time.Second}
var ClimateStateUndefined = AlarmString{Warning, "climate.undefined", "Climate State Undefined", 10 * time.Second, 2 * time.Second}

type MissingDeviceAlarm struct {
	canInterface string
	class        arke.NodeClass
	id           arke.NodeID
}

func (a MissingDeviceAlarm) Flags() AlarmFlags {
	return Warning | AdminOnly
}

func (a MissingDeviceAlarm) Identifier() string {
	return fmt.Sprintf("climate.device_missing.%s.%s.%d", a.canInterface, arke.ClassName(a.class), a.id)
}

func (a MissingDeviceAlarm) Description() string {
	return fmt.Sprintf("Device %s.%s.%d is missing", a.canInterface, arke.ClassName(a.class), a.id)
}

func (a MissingDeviceAlarm) MinUpTime() time.Duration {
	return 1 * time.Minute
}

func (a MissingDeviceAlarm) MinDownTime() time.Duration {
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
	level  AlarmFlags
}

func (a FanAlarm) Identifier() string {
	return "climate.fan." + a.fan
}

func (a FanAlarm) Flags() AlarmFlags {
	return a.level
}

func (a FanAlarm) Description() string {
	status := "aging"
	if a.status == arke.FanStalled {
		status = "stalled"
	}

	return fmt.Sprintf("Fan %s is %s", a.fan, status)
}

func (a FanAlarm) MinUpTime() time.Duration {
	return 2 * time.Minute
}

func (a FanAlarm) MinDownTime() time.Duration {
	return 1 * time.Minute
}

func (a FanAlarm) Fan() string {
	return a.fan
}

func (a FanAlarm) Status() arke.FanStatus {
	return a.status
}

func NewFanAlarm(fan string, s arke.FanStatus, level AlarmFlags) FanAlarm {
	if s == arke.FanOK {
		s = arke.FanAging
	}
	return FanAlarm{fan, s, level & 0x03}
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
	return Warning | AdminOnly
}

func (e DeviceInternalError) MinDownTime() time.Duration {
	return 2 * time.Second
}

func (e DeviceInternalError) MinUpTime() time.Duration {
	return 10 * time.Millisecond
}

func (e DeviceInternalError) Identifier() string {
	return fmt.Sprintf("climate.device_error.%s.%s.%d.%d",
		e.intfName, arke.ClassName(e.class), e.id, e.errorCode)
}

func (e DeviceInternalError) Description() string {
	return fmt.Sprintf("Device %s.%s.%d internal error 0x%04x",
		e.intfName, arke.ClassName(e.class), e.id, e.errorCode)
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
	Identifier     string
	Description    string
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
