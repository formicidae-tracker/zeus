package main

import (
	"github.com/formicidae-tracker/libarke/src-go/arke"
	. "gopkg.in/check.v1"
)

type ArkeDispatcherSuite struct {
	intf *StubRawInterface
	d    ArkeDispatcher
}

var _ = Suite(&ArkeDispatcherSuite{})

func (s *ArkeDispatcherSuite) SetUpTest(c *C) {
	s.intf = NewStubRawInterface()
	s.d = NewArkeDispatcher(s.intf)
}

func (s *ArkeDispatcherSuite) TearDownTest(c *C) {
	c.Check(s.d.Close(), IsNil)
}

func (s *ArkeDispatcherSuite) TestClosesInterface(c *C) {
	c.Check(s.intf.isClosed(), Equals, false)
	c.Check(s.d.Close(), IsNil)
	c.Check(s.intf.isClosed(), Equals, true)
	c.Check(s.d.Close(), IsNil)

	s.SetUpTest(c)
	go s.d.Dispatch()
	c.Check(s.d.Close(), IsNil)
}

func (s *ArkeDispatcherSuite) TestDispatchMessages(c *C) {
	cOne := s.d.Register(1)
	cTwo := s.d.Register(2)
	go s.d.Dispatch()

	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 1) }()
	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 2) }()
	go func() { s.intf.enqueue(&arke.CelaenoSetPoint{}, 3) }()

	_, ok := <-cOne
	c.Check(ok, Equals, true)
	_, ok = <-cTwo
	c.Check(ok, Equals, true)
}
