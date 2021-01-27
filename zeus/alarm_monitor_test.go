package main

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/formicidae-tracker/zeus"
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

func (a testAlarm) Flags() zeus.AlarmFlags {
	return zeus.Warning
}

func (a testAlarm) DeadLine() time.Duration {
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
			waiter(alarms[1].DeadLine()/3, jitterAmount)
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
	c.Check(e.Flags, Equals, alarms[1].Flags())
	c.Check(e.Status, Equals, zeus.AlarmOn)

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
			time.Sleep(alarms[0].DeadLine() / 3)
		}
	}()
	e, ok = <-m.Outbound()
	c.Check(ok, Equals, true)
	c.Check(e.Reason, Equals, alarms[0].Reason())
	c.Check(e.Flags, Equals, alarms[0].Flags())
	c.Check(e.Status, Equals, zeus.AlarmOn)

	e, ok = <-m.Outbound()
	end := time.Now()
	c.Check(ok, Equals, true)
	c.Check(e.Reason, Equals, alarms[0].Reason())
	c.Check(e.Flags, Equals, alarms[0].Flags())
	c.Check(e.Status, Equals, zeus.AlarmOff)

	lasted := end.Sub(start)
	expected := time.Duration(3+repeat-1) * alarms[0].DeadLine() / 3

	c.Check(lasted > expected, Equals, true, Commentf("Lasted %s, expected at least %s", lasted, expected))

	close(quit)
	wg.Wait()
}

func (s *AlarmMonitorSuite) TestReadAlarmLogFile(c *C) {
	testdata := [][]zeus.AlarmEvent{
		nil,
		[]zeus.AlarmEvent{
			zeus.AlarmEvent{
				Zone:   "foo/zone/box",
				Reason: "Ouch `truc`",
				Flags:  zeus.Warning | zeus.InstantNotification,
				Status: zeus.AlarmOn,
				Time:   time.Now().Round(0),
			},
			zeus.AlarmEvent{
				Zone:   "foo/zone/box",
				Reason: "Ouch `truc`",
				Flags:  zeus.Warning | zeus.InstantNotification,
				Status: zeus.AlarmOff,
				Time:   time.Now().Round(0),
			},
		},
	}
	tmpdir, err := ioutil.TempDir("", "read-alarm-file-log")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "log.txt")
	for _, alarms := range testdata {
		am, err := NewFileAlarmReporter(filename)
		if c.Check(err, IsNil) == false {
			continue
		}
		ready := make(chan struct{})
		done := make(chan struct{})
		go func() {
			am.Report(ready)
			close(done)
		}()
		<-ready
		for _, a := range alarms {
			am.AlarmChannel() <- a
		}
		close(am.AlarmChannel())
		<-done
		result, err := ReadAlarmLogFile(filename)
		c.Check(err, IsNil)
		c.Check(result, DeepEquals, alarms)
	}
}
