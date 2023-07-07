package main

import (
	"github.com/formicidae-tracker/libarke/src-go/arke"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	. "gopkg.in/check.v1"
)

type ArkeDispatcherSuite struct {
	intf *StubRawInterface
	d    ArkeDispatcher
	hook *test.Hook
}

var _ = Suite(&ArkeDispatcherSuite{})

func (s *ArkeDispatcherSuite) SetUpTest(c *C) {
	s.intf = NewStubRawInterface()
	s.d = NewArkeDispatcher("can-stub", s.intf)
	_, s.hook = test.NewNullLogger()
	s.d.(*arkeDispatcher).logger.Logger.AddHook(s.hook)
}

func (s *ArkeDispatcherSuite) TearDownTest(c *C) {
	s.hook.Reset()
	c.Check(s.d.Close(), IsNil)
}

func (s *ArkeDispatcherSuite) TestClosesInterface(c *C) {
	c.Check(s.intf.isClosed(), Equals, false)
	c.Check(s.d.Close(), IsNil)
	c.Check(s.intf.isClosed(), Equals, true)
	c.Check(s.d.Close(), IsNil)

	s.SetUpTest(c)
	done := make(chan struct{})
	ready := make(chan struct{})
	go func() {
		s.d.Dispatch(ready)
		close(done)
	}()
	<-ready
	c.Check(s.d.Close(), IsNil)
	<-done
	for _, e := range s.hook.Entries {
		c.Check(e.Level, Equals, logrus.InfoLevel)
		c.Check(e.Data["domain"], Equals, "dispatch/can-stub")
		c.Check(len(e.Data), Equals, 1)
	}
	c.Assert(len(s.hook.Entries), Equals, 2)

	c.Check(s.hook.Entries[0].Message, Equals, "started")
	c.Check(s.hook.Entries[1].Message, Equals, "closed")
}

func (s *ArkeDispatcherSuite) TestDispatchMessages(c *C) {
	cOne := s.d.Register(1)
	cTwo := s.d.Register(2)
	ready := make(chan struct{})
	go s.d.Dispatch(ready)
	<-ready
	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 1) }()
	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 2) }()
	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 3) }()

	_, ok := <-cOne
	c.Check(ok, Equals, true)
	_, ok = <-cTwo
	c.Check(ok, Equals, true)

	for _, e := range s.hook.Entries {
		c.Check(e.Level, Equals, logrus.InfoLevel)
		c.Check(e.Data["domain"], Equals, "dispatch/can-stub")
		c.Check(len(e.Data), Equals, 1)
	}
	c.Assert(len(s.hook.Entries), Equals, 1)

	c.Check(s.hook.Entries[0].Message, Equals, "started")
}

func (s *ArkeDispatcherSuite) TestDoesNotHangUp(c *C) {
	cOne := s.d.Register(1)
	cTwo := s.d.Register(2)
	ready := make(chan struct{})
	go s.d.Dispatch(ready)
	<-ready
	go func() {
		for i := 0; i < 11; i++ {
			s.intf.enqueue(&arke.CelaenoSetPoint{}, 2)
		}
		go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 1) }()
	}()

	_, ok := <-cOne
	c.Check(ok, Equals, true)
	for i := 0; i < 10; i++ {
		<-cTwo
	}

	for _, e := range s.hook.Entries {
		c.Check(e.Data["domain"], Equals, "dispatch/can-stub")
	}
	c.Assert(len(s.hook.Entries), Equals, 2)

	c.Check(len(s.hook.Entries[0].Data), Equals, 1)
	c.Check(s.hook.Entries[0].Level, Equals, logrus.InfoLevel)
	c.Check(s.hook.Entries[0].Message, Equals, "started")

	c.Check(len(s.hook.Entries[1].Data), Equals, 3)
	c.Check(s.hook.Entries[1].Level, Equals, logrus.WarnLevel)
	c.Check(s.hook.Entries[1].Message, Equals, "receiver not ready, dropping message")
	c.Check(s.hook.Entries[1].Data["ID"], Equals, arke.NodeID(2))
	c.Check(s.hook.Entries[1].Data["message"], Equals, "Celaeno.SetPoint{Power: 0}")

}
