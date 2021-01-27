package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type Olympus struct {
	hostname string
	C        chan *C
}

func (h *Olympus) UnregisterZone(zu *zeus.ZoneUnregistration, unused *int) error {
	c := <-h.C
	c.Check(zu.Host, Equals, h.hostname)
	c.Check(zu.Name, Equals, "test-zone")
	*unused = 0
	return nil
}

func (h *Olympus) RegisterZone(zr *zeus.ZoneRegistration, unused *int) error {
	c := <-h.C
	c.Check(zr.Host, Equals, h.hostname)
	c.Check(zr.Name, Equals, "test-zone")
	*unused = 0
	return nil
}

func (h *Olympus) ReportClimate(cr *zeus.NamedClimateReport, unused *int) error {
	c := <-h.C
	c.Check(cr.ZoneIdentifier, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	c.Check(cr.Humidity, Equals, zeus.Humidity(50.0))
	for i := 0; i < 4; i++ {
		c.Check(cr.Temperatures[i], Equals, zeus.Temperature(21))
	}
	*unused = 0
	return nil
}

func (h *Olympus) ReportAlarm(ae *zeus.AlarmEvent, unused *int) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	*unused = 0
	return nil
}

func (h *Olympus) ReportState(ae *zeus.StateReport, unused *int) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	*unused = 0
	return nil
}

const testAddress = ":12345"

type RPCClimateReporterSuite struct {
	Http   *http.Server
	Rpc    *rpc.Server
	H      *Olympus
	Errors chan error
}

var _ = Suite(&RPCClimateReporterSuite{})

func (s *RPCClimateReporterSuite) listenWithError(ready chan struct{}) error {
	l, err := net.Listen("tcp", s.Http.Addr)
	if err != nil {
		return err
	}

	if ready != nil {
		close(ready)
	}

	return s.Http.Serve(l)
}

func (s *RPCClimateReporterSuite) listen(final bool, ready chan struct{}) {
	err := s.listenWithError(ready)

	if err != http.ErrServerClosed {
		s.Errors <- err
	}
	if final == true {
		close(s.Errors)
	} else {
		s.Errors <- nil
	}
}

var baseMux *http.ServeMux

func (s *RPCClimateReporterSuite) SetUpSuite(c *C) {
	baseMux = http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	hostname, err := os.Hostname()
	c.Assert(err, IsNil)
	s.Http = &http.Server{Addr: testAddress}
	s.Rpc = rpc.NewServer()
	s.H = &Olympus{hostname: hostname, C: make(chan *C)}
	s.Rpc.Register(s.H)
	s.Rpc.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	s.Errors = make(chan error)
	ready := make(chan struct{})
	go func() {
		s.listen(false, ready)
	}()
	<-ready
}

func (s *RPCClimateReporterSuite) TearDownSuite(c *C) {
	s.Http.Shutdown(context.Background())
	err, ok := <-s.Errors
	c.Check(ok, Equals, false, Commentf("Got server error: %s", err))
	c.Check(err, IsNil)
	http.DefaultServeMux = baseMux
}

func (s *RPCClimateReporterSuite) TestClimateReport(c *C) {
	go func() { s.H.C <- c }()
	zone := zeus.ZoneClimate{}

	n, err := NewRPCReporter("test-zone", testAddress, zone)
	n.log.SetOutput(bytes.NewBuffer(nil))
	c.Assert(err, IsNil)
	n.MaxAttempts = 2
	n.ReconnectionWindow = 5 * time.Millisecond
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	ready := make(chan struct{})
	go func() {
		n.Report(ready)
		wg.Done()
	}()
	<-ready

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.ReportChannel() <- zeus.ClimateReport{Humidity: 50, Temperatures: []zeus.Temperature{21, 21, 21, 21}}

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.AlarmChannel() <- zeus.AlarmEvent{
		Zone:   zeus.ZoneIdentifier(s.H.hostname, "test-zone"),
		Reason: "foo",
		Flags:  zeus.Warning,
		Status: zeus.AlarmOn,
		Time:   time.Now(),
	}

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.StateChannel() <- zeus.StateReport{
		Zone: zeus.ZoneIdentifier(s.H.hostname, "test-zone"),
	}

	s.Http.Shutdown(context.Background())
	err, ok := <-s.Errors
	c.Check(ok, Equals, true)
	c.Check(err, IsNil)

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.StateChannel() <- zeus.StateReport{
		Zone: zeus.ZoneIdentifier(s.H.hostname, "test-zone"),
	}

	time.Sleep(time.Duration(n.MaxAttempts+100) * n.ReconnectionWindow)
	ready = make(chan struct{})
	go s.listen(true, ready)
	<-ready
	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	close(n.ReportChannel())
	close(n.AlarmChannel())
	close(n.StateChannel())
	wg.Wait()
}
