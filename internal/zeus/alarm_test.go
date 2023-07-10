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
		{NewFanAlarm("foo", arke.FanAging, Warning), 10 * time.Minute},
		{NewMissingDeviceAlarm("foo", arke.ZeusClass, 1), 5 * HeartBeatPeriod},
		{NewDeviceInternalError("foo", arke.ZeusClass, 1, 43), 2 * time.Second},
	}

	for _, d := range testdata {
		c.Check(d.Alarm.MinDownTime(), Equals, d.Expected)
	}

}

func (s *AlarmSuite) TestData(c *C) {
	testdata := []struct {
		Alarm               Alarm
		ExpectedIdentifier  string
		ExpectedDescription string
		ExpectedFlags       AlarmFlags
	}{
		{
			Alarm:               WaterLevelWarning,
			ExpectedIdentifier:  "climate.water_level",
			ExpectedDescription: "Water tank level is low",
			ExpectedFlags:       Emergency,
		},
		{
			Alarm:               WaterLevelCritical,
			ExpectedIdentifier:  "climate.water_level",
			ExpectedDescription: "Water tank is empty",
			ExpectedFlags:       Failure,
		},
		{
			Alarm:               WaterLevelUnreadable,
			ExpectedIdentifier:  "climate.water_sensor",
			ExpectedDescription: "Celaeno cannot determine water tank level",
			ExpectedFlags:       Failure,
		},
		{
			Alarm:               HumidityUnreachable,
			ExpectedIdentifier:  "climate.humidity.unreachable",
			ExpectedDescription: "Cannot reach desired humidity",
			ExpectedFlags:       Warning,
		},
		{
			Alarm:               TemperatureUnreachable,
			ExpectedIdentifier:  "climate.temperature.unreachable",
			ExpectedDescription: "Cannot reach desired temperature",
			ExpectedFlags:       Warning,
		},
		{
			Alarm:               OutOfBound[Humidity](40, 80),
			ExpectedIdentifier:  "climate.humidity.out_of_bounds",
			ExpectedDescription: "Humidity is outside of boundaries ( [40.0 ; 80.0] % R.H. )",
			ExpectedFlags:       Emergency,
		},
		{
			Alarm:               OutOfBound[Temperature](10, 25),
			ExpectedIdentifier:  "climate.temperature.out_of_bounds",
			ExpectedDescription: "Temperature is outside of boundaries ( [10.0 ; 25.0] °C )",
			ExpectedFlags:       Emergency,
		},
		{
			Alarm:               SensorReadoutIssue,
			ExpectedIdentifier:  "climate.sensor.readout",
			ExpectedDescription: "Cannot read sensors",
			ExpectedFlags:       Failure,
		},
		{
			Alarm:               NewFanAlarm("foo", arke.FanStalled, Emergency),
			ExpectedIdentifier:  "climate.fan.foo",
			ExpectedDescription: "Fan foo is stalled",
			ExpectedFlags:       Emergency,
		},
		{
			Alarm:               NewFanAlarm("bar", arke.FanAging, Warning),
			ExpectedIdentifier:  "climate.fan.bar",
			ExpectedDescription: "Fan bar is aging",
			ExpectedFlags:       Warning,
		},
		{
			Alarm:               NewFanAlarm("baz", arke.FanOK, Warning),
			ExpectedIdentifier:  "climate.fan.baz",
			ExpectedDescription: "Fan baz is aging",
			ExpectedFlags:       Warning,
		},
		{
			Alarm:               NewMissingDeviceAlarm("vcan0", arke.ZeusClass, 1),
			ExpectedIdentifier:  "climate.device_missing.vcan0.Zeus.1",
			ExpectedDescription: "Device vcan0.Zeus.1 is missing",
			ExpectedFlags:       Warning | AdminOnly,
		},
		{
			Alarm:               NewMissingDeviceAlarm("vcan0", arke.CelaenoClass, 1),
			ExpectedIdentifier:  "climate.device_missing.vcan0.Celaeno.1",
			ExpectedDescription: "Device vcan0.Celaeno.1 is missing",
			ExpectedFlags:       Warning | AdminOnly,
		},
		{
			Alarm:               NewDeviceInternalError("vcan0", arke.ZeusClass, 1, 0x42),
			ExpectedIdentifier:  "climate.device_error.vcan0.Zeus.1.66",
			ExpectedDescription: "Device vcan0.Zeus.1 internal error 0x0042",
			ExpectedFlags:       Warning | AdminOnly,
		},
	}

	for _, d := range testdata {
		comment := Commentf("identifier: %s, description: %s", d.ExpectedIdentifier, d.ExpectedDescription)
		c.Check(d.Alarm.Identifier(), Equals, d.ExpectedIdentifier, comment)
		c.Check(d.Alarm.Description(), Equals, d.ExpectedDescription, comment)
		c.Check(d.Alarm.Flags(), Equals, d.ExpectedFlags, comment)
	}

}

func (s *AlarmSuite) TestFanAlarm(c *C) {
	testdata := []struct {
		Alarm          FanAlarm
		ExpectedStatus arke.FanStatus
		ExpectedFan    string
	}{
		{NewFanAlarm("foo", arke.FanAging, Warning), arke.FanAging, "foo"},
		{NewFanAlarm("bar", arke.FanStalled, Warning), arke.FanStalled, "bar"},
		{NewFanAlarm("baz", arke.FanOK, Warning), arke.FanAging, "baz"},
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
		{Warning, 1},
		{Emergency, 2},
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
