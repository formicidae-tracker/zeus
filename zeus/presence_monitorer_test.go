package main

import (
	"bytes"
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type PresenceMonitorerSuite struct {
	intf *StubRawInterface
	m    PresenceMonitorer
}

var _ = Suite(&PresenceMonitorerSuite{})

func (s *PresenceMonitorerSuite) SetUpTest(c *C) {
	s.intf = NewStubRawInterface()
	s.m = NewPresenceMonitorer("test-can", s.intf)
	s.m.(*presenceMonitorer).logger.SetOutput(bytes.NewBuffer(nil))
	s.m.(*presenceMonitorer).HeartBeatPeriod = 1 * time.Millisecond
}

func (s *PresenceMonitorerSuite) TearDownTest(c *C) {
	c.Assert(s.intf.isClosed(), Equals, false)
	s.intf.Close()
}

func (s *PresenceMonitorerSuite) TestClose(c *C) {
	c.Check(s.m.Close(), ErrorMatches, "Already closed")
	go s.m.Monitor(nil, nil)
	time.Sleep(1 * time.Millisecond)
	c.Check(s.m.Close(), IsNil)
}

func (s *PresenceMonitorerSuite) TestAlarmsMissingDevices(c *C) {
	alarms := make(chan zeus.Alarm)
	devices := []DeviceDefinition{
		DeviceDefinition{Class: arke.ZeusClass, ID: 1},
		DeviceDefinition{Class: arke.CelaenoClass, ID: 1},
		DeviceDefinition{Class: arke.HeliosClass, ID: 1},
	}
	go func() {
		s.m.Ping(arke.ZeusClass, 1)
		s.m.Ping(arke.CelaenoClass, 2)
		s.m.Ping(arke.HeliosClass, 1)
	}()
	go s.m.Monitor(devices, alarms)
	a, ok := <-alarms
	c.Check(ok, Equals, true)
	c.Check(a, DeepEquals, zeus.NewMissingDeviceAlarm("test-can", arke.CelaenoClass, 1))
	c.Check(s.m.Close(), IsNil)
}
