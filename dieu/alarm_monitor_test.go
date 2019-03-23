package main

import (
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/formicidae-tracker/dieu"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type AlarmMonitorSuite struct {
	Hostname string
}

var _ = Suite(&AlarmMonitorSuite{})

func (s *AlarmMonitorSuite) SetUpSuite(c *C) {
	var err error
	s.Hostname, err = os.Hostname()
	c.Assert(err, IsNil)
}

type testAlarm string

func (a testAlarm) Reason() string {
	return string(a)
}

func (a testAlarm) Priority() dieu.Priority {
	return dieu.Warning
}

func (a testAlarm) RepeatPeriod() time.Duration {
	return 5 * time.Millisecond
}

func (s *AlarmMonitorSuite) TestName(c *C) {
	testName := "test-zone"
	m, err := NewAlarmMonitor(testName)
	c.Assert(err, IsNil)
	c.Check(m.Name(), Equals, path.Join(s.Hostname, "zone", testName))
}

func (s *AlarmMonitorSuite) TestMonitor(c *C) {
	m, err := NewAlarmMonitor("test-zone")
	c.Assert(err, IsNil)
	wg := sync.WaitGroup{}

	alarms := []testAlarm{"once", "recurring"}

	quit := make(chan struct{})

	jitterAmount := 0.2 // 20 %

	waiter := func(period time.Duration, jitter float64) {
		// adds a little bit of jitter
		mult := 1.0 + 2*jitter*rand.Float64() - jitter
		time.Sleep(time.Duration(mult*float64(period.Nanoseconds())) * time.Nanosecond)

	}

	go func() {
		wg.Add(1)
		defer func() {
			close(m.Inbound())
			wg.Done()
		}()

		for {
			waiter(alarms[1].RepeatPeriod(), jitterAmount)
			select {
			case <-quit:
				return
			default:
			}
			m.Inbound() <- alarms[1]
		}
	}()

	go func() {
		wg.Add(1)
		m.Monitor()
		wg.Done()
	}()

	e, ok := <-m.Outbound()
	c.Check(ok, Equals, true)
	c.Check(e.Reason, Equals, alarms[1].Reason())
	c.Check(e.Priority, Equals, alarms[1].Priority())
	c.Check(e.Status, Equals, dieu.AlarmOn)

	start := time.Now()
	repeat := 10
	go func() {
		for i := 0; i < repeat; i++ {
			select {
			case <-quit:
				return
			default:
			}
			m.Inbound() <- alarms[0]
			time.Sleep(alarms[0].RepeatPeriod())
		}
	}()
	e, ok = <-m.Outbound()
	c.Check(ok, Equals, true)
	c.Check(e.Reason, Equals, alarms[0].Reason())
	c.Check(e.Priority, Equals, alarms[0].Priority())
	c.Check(e.Status, Equals, dieu.AlarmOn)

	e, ok = <-m.Outbound()
	end := time.Now()
	c.Check(ok, Equals, true)
	c.Check(e.Reason, Equals, alarms[0].Reason())
	c.Check(e.Priority, Equals, alarms[0].Priority())
	c.Check(e.Status, Equals, dieu.AlarmOff)

	lasted := end.Sub(start)
	expected := time.Duration(3+repeat-1) * alarms[0].RepeatPeriod()

	c.Check(lasted > expected, Equals, true, Commentf("Lasted %s, expected at least %s", lasted, expected))

	close(quit)
	wg.Wait()
}
