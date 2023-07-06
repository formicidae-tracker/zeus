package main

import (
	"io"
	"log"
	"net"

	olympuspb "github.com/formicidae-tracker/olympus/api"
	"github.com/formicidae-tracker/zeus/internal/zeus"
	"google.golang.org/grpc"
	. "gopkg.in/check.v1"
)

type olympusStub struct {
	olympuspb.UnimplementedOlympusServer

	received chan *olympuspb.ClimateUpStream
}

func (o *olympusStub) Climate(s olympuspb.Olympus_ClimateServer) error {
	defer close(o.received)
	for {
		m, err := s.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.Send(&olympuspb.ClimateDownStream{})
		o.received <- m
	}
}

type RPCClimateReporterSuite struct {
	server  *grpc.Server
	olympus *olympusStub
	errors  chan error
}

var _ = Suite(&RPCClimateReporterSuite{})

func (s *RPCClimateReporterSuite) SetUpTest(c *C) {
	s.server = grpc.NewServer(olympuspb.DefaultServerOptions...)
	s.olympus = &olympusStub{received: make(chan *olympuspb.ClimateUpStream, 10)}
	olympuspb.RegisterOlympusServer(s.server, s.olympus)
	s.errors = make(chan error)
	go func() {
		var err error
		defer func() {
			s.errors <- err
			close(s.errors)
		}()
		l, err := net.Listen("tcp", "localhost:12345")
		if err != nil {
			return
		}

		err = s.server.Serve(l)
	}()
}

func (s *RPCClimateReporterSuite) TearDownTest(c *C) {
	s.server.GracefulStop()
	err, ok := <-s.errors
	c.Check(ok, Equals, true)
	err, ok = <-s.errors
	c.Check(err, IsNil)
	c.Check(ok, Equals, false)
}

func (s *RPCClimateReporterSuite) TestEnd2End(c *C) {
	r, err := NewRPCReporter(RPCReporterOptions{
		zone:           "box",
		olympusAddress: "localhost:12345",
		host:           "myself",
		runner:         nil,
	})
	r.connected = make(chan bool)
	c.Assert(err, IsNil)
	ready := make(chan struct{})
	go r.Report(ready)
	<-ready

	m, ok := <-s.olympus.received
	c.Check(m.Declaration, Not(IsNil))
	c.Assert(ok, Equals, true)

	<-r.connected

	log.Printf("sending climate report")
	r.ReportChannel() <- zeus.ClimateReport{}
	m, ok = <-s.olympus.received
	c.Check(m.Reports, Not(IsNil))
	c.Assert(ok, Equals, true)

	log.Printf("sending alarm event")
	r.AlarmChannel() <- zeus.AlarmEvent{}
	m, ok = <-s.olympus.received
	c.Check(m.Alarms, Not(IsNil))
	c.Assert(ok, Equals, true)

	log.Printf("sending climate target")
	r.TargetChannel() <- zeus.ClimateTarget{}
	m, ok = <-s.olympus.received
	c.Check(m.Target, Not(IsNil))
	c.Assert(ok, Equals, true)

	log.Printf("closing")
	close(r.ReportChannel())
	close(r.AlarmChannel())
	close(r.TargetChannel())
	m, ok = <-s.olympus.received
	c.Check(m, IsNil)
	c.Assert(ok, Equals, false)

}
