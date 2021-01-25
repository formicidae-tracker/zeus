package zeus

import (
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	. "gopkg.in/check.v1"
)

type AlarmSuite struct{}

var _ = Suite(&AlarmSuite{})

func (s *AlarmSuite) TestRepeatPeriod(c *C) {
	testdata := []struct {
		Alarm    Alarm
		Expected time.Duration
	}{
		{AlarmString{}, 0},
		{NewFanAlarm("foo", arke.FanAging), 10 * time.Minute},
		{NewMissingDeviceAlarm("foo", arke.ZeusClass, 1), 5 * HeartBeatPeriod},
		{NewDeviceInternalError("foo", arke.ZeusClass, 1, 43), 2 * time.Second},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.DeadLine(), Equals, d.Expected)
	}

}

func (s *AlarmSuite) TestData(c *C) {
	testdata := []struct {
		Alarm          Alarm
		ExpectedReason string
		ExpectedFlags  AlarmFlags
	}{
		{WaterLevelWarning, "Celaeno water level is low", Warning | InstantNotification},
		{WaterLevelCritical, "Celaeno is empty", Emergency | InstantNotification},
		{WaterLevelUnreadable, "Celaeno water level is unreadable", Emergency | InstantNotification},
		{HumidityUnreachable, "Cannot reach desired humidity", Warning},
		{TemperatureUnreachable, "Cannot reach desired temperature", Warning},
		{HumidityOutOfBound, "Humidity is outside of boundaries", Emergency | InstantNotification},
		{TemperatureOutOfBound, "Temperature is outside of boundaries", Emergency | InstantNotification},
		{SensorReadoutIssue, "Cannot read sensors", Emergency},
		{NewFanAlarm("foo", arke.FanStalled), "Fan foo is stalled", Emergency},
		{NewFanAlarm("bar", arke.FanAging), "Fan bar is aging", Warning},
		{NewFanAlarm("baz", arke.FanOK), "Fan baz is aging", Warning},
		{NewMissingDeviceAlarm("vcan0", arke.ZeusClass, 1), "Device vcan0.Zeus.1 is missing", Emergency | InstantNotification},
		{NewDeviceInternalError("vcan0", arke.ZeusClass, 1, 0x42), "Device vcan0.Zeus.1 internal error 0x0042", Warning},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.Reason(), Equals, d.ExpectedReason)
		c.Check(d.Alarm.Flags(), Equals, d.ExpectedFlags)
	}

}

func (s *AlarmSuite) TestFanAlarm(c *C) {
	testdata := []struct {
		Alarm          FanAlarm
		ExpectedStatus arke.FanStatus
		ExpectedFan    string
	}{
		{NewFanAlarm("foo", arke.FanAging), arke.FanAging, "foo"},
		{NewFanAlarm("bar", arke.FanStalled), arke.FanStalled, "bar"},
		{NewFanAlarm("baz", arke.FanOK), arke.FanAging, "baz"},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.Fan(), Equals, d.ExpectedFan)
		c.Check(d.Alarm.Status(), Equals, d.ExpectedStatus)
	}
}

func (s *AlarmSuite) TestMisisngDeviceAlarm(c *C) {
	testdata := []struct {
		Alarm             MissingDeviceAlarm
		ExpectedInterface string
		ExpectedClass     arke.NodeClass
		ExpectedID        arke.NodeID
	}{
		{NewMissingDeviceAlarm("vcan0", arke.CelaenoClass, 1), "vcan0", arke.CelaenoClass, 1},
		{NewMissingDeviceAlarm("vcan0", arke.CelaenoClass, 0), "vcan0", arke.CelaenoClass, 1},
		{NewMissingDeviceAlarm("vcan0", arke.CelaenoClass, 8), "vcan0", arke.CelaenoClass, 7},
	}

	for _, d := range testdata {
		intf, class, ID := d.Alarm.Device()
		c.Check(intf, Equals, d.ExpectedInterface)
		c.Check(class, Equals, d.ExpectedClass)
		c.Check(ID, Equals, d.ExpectedID)
	}
}

func (s *AlarmSuite) TestLevelMapping(c *C) {
	testdata := []struct {
		Flags AlarmFlags
		Level int
	}{
		{Warning, 1},
		{Emergency, 2},
		{Warning | InstantNotification, 1},
		{Emergency | InstantNotification, 2},
	}

	for _, d := range testdata {
		c.Check(MapPriority(d.Flags), Equals, d.Level)
	}
}

func (s *AlarmSuite) TestDeviceInternalError(c *C) {
	testdata := []struct {
		Alarm             DeviceInternalError
		ExpectedInterface string
		ExpectedClass     arke.NodeClass
		ExpectedID        arke.NodeID
		ExpectedError     uint16
	}{
		{NewDeviceInternalError("vcan0", arke.CelaenoClass, 1, 0x42), "vcan0", arke.CelaenoClass, 1, 0x42},
	}

	for _, d := range testdata {
		intf, class, ID := d.Alarm.Device()
		c.Check(intf, Equals, d.ExpectedInterface)
		c.Check(class, Equals, d.ExpectedClass)
		c.Check(ID, Equals, d.ExpectedID)
		c.Check(d.Alarm.ErrorCode(), Equals, d.ExpectedError)
	}
}
