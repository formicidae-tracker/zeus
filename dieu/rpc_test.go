package main

import (
	"context"
	"net/http"
	"net/rpc"
	"os"
	"sync"

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
	c.Check(cr.ZoneIdentifier, Equals, h.hostname+"/zone/test-zone")
	c.Check(cr.Humidity, Equals, dieu.Humidity(50.0))
	for i := 0; i < 4; i++ {
		c.Check(cr.Temperatures[i], Equals, dieu.Temperature(21))
	}
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

func (s *RPCClimateReporterSuite) SetUpSuite(c *C) {
	hostname, err := os.Hostname()
	c.Assert(err, IsNil)
	s.Http = &http.Server{Addr: testAddress}
	s.Rpc = rpc.NewServer()
	s.H = &Hermes{hostname: hostname, C: make(chan *C)}
	s.Rpc.Register(s.H)
	s.Rpc.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	s.Errors = make(chan error)
	go func() {
		err := s.Http.ListenAndServe()
		if err != http.ErrServerClosed {
			s.Errors <- err
		}
		close(s.Errors)
	}()
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
	n, err := NewRPCReporter("test-zone", testAddress, zone)
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		n.Report(&wg)
	}()

	go func() { s.H.C <- c }()
	n.ReportChannel() <- dieu.ClimateReport{Humidity: 50, Temperatures: [4]dieu.Temperature{21, 21, 21, 21}}

	go func() { s.H.C <- c }()
	close(n.ReportChannel())
	close(n.AlarmChannel())
	close(n.StateChannel())
	wg.Wait()
}
