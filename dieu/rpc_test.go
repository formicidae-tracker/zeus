package main

import (
	"bytes"
	"context"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"git.tuleu.science/fort/dieu"
	. "gopkg.in/check.v1"
)

type Hermes struct {
	hostname string
	C        chan *C
}

func (h *Hermes) UnregisterZone(zu *dieu.ZoneUnregistration, err *dieu.HermesError) error {
	c := <-h.C
	c.Check(zu.Host, Equals, h.hostname)
	c.Check(zu.Name, Equals, "test-zone")
	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) RegisterZone(zr *dieu.ZoneRegistration, err *dieu.HermesError) error {
	c := <-h.C
	c.Check(zr.Host, Equals, h.hostname)
	c.Check(zr.Name, Equals, "test-zone")
	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportClimate(cr *dieu.NamedClimateReport, err *dieu.HermesError) error {
	c := <-h.C
	c.Check(cr.ZoneIdentifier, Equals, dieu.ZoneIdentifier(h.hostname, "test-zone"))
	c.Check(cr.Humidity, Equals, dieu.Humidity(50.0))
	for i := 0; i < 4; i++ {
		c.Check(cr.Temperatures[i], Equals, dieu.Temperature(21))
	}
	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportAlarm(ae *dieu.AlarmEvent, err *dieu.HermesError) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, dieu.ZoneIdentifier(h.hostname, "test-zone"))
	*err = dieu.HermesError("")
	return nil
}

func (h *Hermes) ReportState(ae *dieu.StateReport, err *dieu.HermesError) error {
	c := <-h.C
	c.Check(ae.Zone, Equals, dieu.ZoneIdentifier(h.hostname, "test-zone"))
	*err = dieu.HermesError("")
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
	err, ok := <-s.Errors
	c.Check(ok, Equals, false)
	c.Check(err, IsNil)
}

func (s *RPCClimateReporterSuite) TestClimateReport(c *C) {
	go func() { s.H.C <- c }()
	zone := dieu.Zone{}

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
	n.ReportChannel() <- dieu.ClimateReport{Humidity: 50, Temperatures: [4]dieu.Temperature{21, 21, 21, 21}}

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.AlarmChannel() <- dieu.AlarmEvent{
		Zone:     dieu.ZoneIdentifier(s.H.hostname, "test-zone"),
		Reason:   "foo",
		Priority: dieu.Warning,
		Status:   dieu.AlarmOn,
		Time:     time.Now(),
	}

	wg.Add(1)
	go func() {
		s.H.C <- c
		wg.Done()
	}()
	n.StateChannel() <- dieu.StateReport{
		Zone: dieu.ZoneIdentifier(s.H.hostname, "test-zone"),
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
	n.StateChannel() <- dieu.StateReport{
		Zone: dieu.ZoneIdentifier(s.H.hostname, "test-zone"),
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
