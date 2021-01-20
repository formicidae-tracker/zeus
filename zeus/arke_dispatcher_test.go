package main

import (
	"bytes"

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
	s.d = NewArkeDispatcher("can-stub", s.intf)
	s.d.(*arkeDispatcher).logger.SetOutput(bytes.NewBuffer(nil))
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
	done := make(chan struct{})
	go func() {
		s.d.Dispatch()
		close(done)
	}()
	c.Check(s.d.Close(), IsNil)
	<-done
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

func (s *ArkeDispatcherSuite) TestDoesNotHangUp(c *C) {
	cOne := s.d.Register(1)
	cTwo := s.d.Register(2)
	go s.d.Dispatch()

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
}
