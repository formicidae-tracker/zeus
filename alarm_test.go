package dieu

import (
	"time"

	"git.tuleu.science/fort/libarke/src-go/arke"
	. "gopkg.in/check.v1"
)

type AlarmSuite struct{}

var _ = Suite(&AlarmSuite{})

func (s *AlarmSuite) TestRepeatPeriod(c *C) {
	testdata := []struct {
		Alarm    Alarm
		Expected time.Duration
	}{
		{AlarmString{}, 500 * time.Millisecond},
		{NewFanAlarm("foo", arke.FanAging), 500 * time.Millisecond},
		{NewMissingDeviceAlarm("foo", arke.ZeusClass, 1), HeartBeatPeriod},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.RepeatPeriod(), Equals, d.Expected)
	}

}

func (s *AlarmSuite) TestData(c *C) {
	testdata := []struct {
		Alarm            Alarm
		ExpectedReason   string
		ExpectedPriority Priority
	}{
		{WaterLevelWarning, "Celaeno water level is low", Warning},
		{WaterLevelCritical, "Celaeno is empty", Emergency},
		{WaterLevelUnreadable, "Celaeno water level is unreadable", Emergency},
		{HumidityUnreachable, "Cannot reach desired humidity", Warning},
		{TemperatureUnreachable, "Cannot reach desired temperature", Warning},
		{HumidityOutOfBound, "Humidity is outside of boundaries", Emergency},
		{TemperatureOutOfBound, "Temperature is outside of boundaries", Emergency},
		{SensorReadoutIssue, "Cannot read sensors", Emergency},
		{NewFanAlarm("foo", arke.FanStalled), "Fan foo is stalled", Emergency},
		{NewFanAlarm("bar", arke.FanAging), "Fan bar is aging", Warning},
		{NewFanAlarm("baz", arke.FanOK), "Fan baz is aging", Warning},
		{NewMissingDeviceAlarm("vcan0", arke.ZeusClass, 1), "Device vcan0.Zeus.1 is missing", Emergency},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.Reason(), Equals, d.ExpectedReason)
		c.Check(d.Alarm.Priority(), Equals, d.ExpectedPriority)
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
		Priority Priority
		Level    int
	}{
		{Warning, 1},
		{Emergency, 2},
		{100, 2},
		{-100, 2},
	}

	for _, d := range testdata {
		c.Check(MapPriority(d.Priority), Equals, d.Level)
	}
}
