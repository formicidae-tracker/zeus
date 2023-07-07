package main

import (
	"time"

	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	. "gopkg.in/check.v1"
)

type PresenceMonitorerSuite struct {
	intf *StubRawInterface
	m    PresenceMonitorer
	hook *test.Hook
}

var _ = Suite(&PresenceMonitorerSuite{})

func (s *PresenceMonitorerSuite) SetUpTest(c *C) {
	s.intf = NewStubRawInterface()
	s.m = NewPresenceMonitorer("test-can", s.intf)
	_, s.hook = test.NewNullLogger()
	s.m.(*presenceMonitorer).logger.Logger.AddHook(s.hook)
	s.m.(*presenceMonitorer).HeartBeatPeriod = 1 * time.Millisecond
}

func (s *PresenceMonitorerSuite) TearDownTest(c *C) {
	s.hook.Reset()
	c.Assert(s.intf.isClosed(), Equals, false)
	s.intf.Close()
}

func (s *PresenceMonitorerSuite) TestClose(c *C) {
	c.Check(s.m.Close(), ErrorMatches, "Already closed")
	ready := make(chan struct{})
	go s.m.Monitor(nil, nil, ready)
	<-ready
	time.Sleep(1 * time.Millisecond)
	c.Check(s.m.Close(), IsNil)
}

func (s *PresenceMonitorerSuite) TestAlarmsMissingDevices(c *C) {
	alarms := make(chan zeus.Alarm)
	devices := []DeviceDefinition{
		{Class: arke.ZeusClass, ID: 1},
		{Class: arke.CelaenoClass, ID: 1},
		{Class: arke.HeliosClass, ID: 1},
	}
	go func() {
		s.m.Ping(arke.ZeusClass, 1)
		s.m.Ping(arke.CelaenoClass, 2)
		s.m.Ping(arke.HeliosClass, 1)
	}()
	ready := make(chan struct{})
	go s.m.Monitor(devices, alarms, ready)
	<-ready
	a, ok := <-alarms

	c.Check(ok, Equals, true)
	c.Check(a, DeepEquals, zeus.NewMissingDeviceAlarm("test-can", arke.CelaenoClass, 1))
	c.Check(s.m.Close(), IsNil)

	c.Assert(len(s.hook.Entries) >= 1, Equals, true)
	e := s.hook.Entries[0]
	c.Check(e.Level, Equals, logrus.WarnLevel)
	c.Check(e.Data["device"], Equals, DeviceDefinition{Class: arke.CelaenoClass, ID: 2})
	c.Check(e.Data["domain"], Equals, "monitor/test-can")
	c.Check(e.Message, Equals, "unmonitored device")

}
