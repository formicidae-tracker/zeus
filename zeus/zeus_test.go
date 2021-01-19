package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	socketcan "github.com/atuleu/golang-socketcan"
	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type ZeusSuite struct {
	interfaces map[string][]*StubRawInterface
	oldmux     *http.ServeMux
	zeus       *Zeus
}

var _ = Suite(&ZeusSuite{})

func (s *ZeusSuite) SetUpTest(c *C) {
	s.oldmux = http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	s.interfaces = map[string][]*StubRawInterface{
		"slcan0": nil,
		"slcan1": nil,
	}
	var err error
	s.zeus, err = OpenZeus(Config{
		Interfaces: map[string]string{
			"slcan0": "foo",
			"slcan1": "bar",
		},
		Zones: map[string]ZoneDefinition{
			"nest": ZoneDefinition{
				CANInterface: "slcan0",
				DevicesID:    1,
			},
			"foraging": ZoneDefinition{
				CANInterface: "slcan0",
				DevicesID:    2,
			},
			"tunnel": ZoneDefinition{
				CANInterface: "slcan1",
				DevicesID:    1},
		},
	})
	c.Check(err, IsNil)
	s.zeus.intfFactory = s.interfaceFactory()
}

func (s *ZeusSuite) TearDownTest(c *C) {
	http.DefaultServeMux = s.oldmux
}

func (s *ZeusSuite) interfaceFactory() func(string) (socketcan.RawInterface, error) {
	return func(name string) (socketcan.RawInterface, error) {
		opened, ok := s.interfaces[name]
		if ok == false {
			return nil, fmt.Errorf("No such device %s", name)
		}
		intf := NewStubRawInterface()
		s.interfaces[name] = append(opened, intf)
		return intf, nil
	}
}

func (s *ZeusSuite) TestWrongConfig(c *C) {
	z, err := OpenZeus(Config{
		Zones: map[string]ZoneDefinition{
			"box": ZoneDefinition{
				CANInterface: "slcan0",
				DevicesID:    1,
			},
		},
	})
	c.Check(z, IsNil)
	c.Check(err, ErrorMatches, "Invalid config:.*")
}

func (s *ZeusSuite) TestShutdown(c *C) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s.zeus.run()
		wg.Done()
	}()

	time.Sleep(200 * time.Millisecond)
	s.zeus.shutdown()
	wg.Wait()

}

func (s *ZeusSuite) TestStartStop(c *C) {
	c.Check(s.zeus.isRunning(), Equals, false)
	c.Check(s.zeus.stopClimate(), ErrorMatches, "Not running")
	c.Check(s.zeus.startClimate(zeus.SeasonFile{
		Zones: map[string]zeus.ZoneClimate{
			"nest": zeus.ZoneClimate{},
		},
	}), IsNil)
	c.Check(s.zeus.isRunning(), Equals, true)
	c.Check(s.zeus.startClimate(zeus.SeasonFile{}), ErrorMatches, "Already started")
	c.Check(s.zeus.stopClimate(), IsNil)
}
