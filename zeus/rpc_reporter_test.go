package main

import (
	"bytes"
	"context"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type Hermes struct {
	hostname string
	C        chan *C
}

func (h *Hermes) UnregisterZone(zu *zeus.ZoneUnregistration, err *zeus.ZeusError) error {
	c := <-h.C
	c.Check(zu.Host, Equals, h.hostname)
	c.Check(zu.Name, Equals, "test-zone")
	*err = zeus.ZeusError("")
	return nil
}

func (h *Hermes) RegisterZone(zr *zeus.ZoneRegistration, err *zeus.ZeusError) error {
	c := <-h.C
	c.Check(zr.Host, Equals, h.hostname)
	c.Check(zr.Name, Equals, "test-zone")
	*err = zeus.ZeusError("")
	return nil
}

func (h *Hermes) ReportClimate(cr *zeus.NamedClimateReport, err *zeus.ZeusError) error {
	c := <-h.C
	c.Check(cr.ZoneIdentifier, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	c.Check(cr.Humidity, Equals, zeus.Humidity(50.0))
	for i := 0; i < 4; i++ {
		c.Check(cr.Temperatures[i], Equals, zeus.Temperature(21))
	}
	*err = zeus.ZeusError("")
	return nil
}

func (h *Hermes) ReportAlarm(ae *zeus.AlarmEvent, err *zeus.ZeusError) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	*err = zeus.ZeusError("")
	return nil
}

func (h *Hermes) ReportState(ae *zeus.StateReport, err *zeus.ZeusError) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, zeus.ZoneIdentifier(h.hostname, "test-zone"))
	*err = zeus.ZeusError("")
	return nil
}

const testAddress = "localhost:12345"

type RPCClimateReporterSuite struct {
	Http   *http.Server
	Rpc    *rpc.Server
	H      *Hermes
	Errors chan error
}

var _ = Suite(&RPCClimateReporterSuite{})

func (s *RPCClimateReporterSuite) listen(final bool) {
	err := s.Http.ListenAndServe()
	if err != http.ErrServerClosed {
		s.Errors <- err
	}
	if final == true {
		close(s.Errors)
	} else {
		s.Errors <- nil
	}
}

func (s *RPCClimateReporterSuite) SetUpSuite(c *C) {
	hostname, err := os.Hostname()
	c.Assert(err, IsNil)
	s.Http = &http.Server{Addr: testAddress}
	s.Rpc = rpc.NewServer()
	s.H = &Hermes{hostname: hostname, C: make(chan *C)}
	s.Rpc.Register(s.H)
	s.Rpc.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	s.Errors = make(chan error)
	go s.listen(false)
}

func (s *RPCClimateReporterSuite) TearDownSuite(c *C) {
	s.Http.Shutdown(context.Background())
	//	err, ok := <-s.Errors
	//c.Check(ok, Equals, false)
	//	c.Check(err, IsNil)
}

func (s *RPCClimateReporterSuite) TestClimateReport(c *C) {
	return
	go func() { s.H.C <- c }()
	zone := zeus.Zone{}

	n, err := NewRPCReporter("test-zone", testAddress, zone, bytes.NewBuffer(nil))
	n.MaxAttempts = 2
	n.ReconnectionWindow = 5 * time.Millisecond
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		n.Report(&wg)
	}()

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.ReportChannel() <- zeus.ClimateReport{Humidity: 50, Temperatures: [4]zeus.Temperature{21, 21, 21, 21}}

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.AlarmChannel() <- zeus.AlarmEvent{
		Zone:     zeus.ZoneIdentifier(s.H.hostname, "test-zone"),
		Reason:   "foo",
		Priority: zeus.Warning,
		Status:   zeus.AlarmOn,
		Time:     time.Now(),
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
	go s.listen(true)

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
