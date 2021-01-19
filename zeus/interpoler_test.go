package main

import (
	"bytes"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type InterpolationManagerSuite struct {
}

var _ = Suite(&InterpolationManagerSuite{})

func (s *InterpolationManagerSuite) TestInterpolationReportsSanitized(c *C) {

	states := []zeus.State{
		zeus.State{
			Name:         "single and only one",
			Temperature:  22.0,
			Humidity:     50.0,
			Wind:         100.0,
			VisibleLight: zeus.UndefinedLight,
			UVLight:      zeus.UndefinedLight,
		},
	}

	reports := make(chan zeus.StateReport, 1)

	m, err := NewInterpoler("test-zone", states, []zeus.Transition{}, []capability{}, reports, bytes.NewBuffer(nil))
	c.Assert(err, IsNil)

	m.period = 1 * time.Millisecond
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Interpolate()
		wg.Done()
	}()

	r, ok := <-reports
	c.Assert(r.Current, Not(IsNil))
	c.Check(ok, Equals, true)
	c.Check(r.Current.Temperature, Equals, states[0].Temperature)
	c.Check(r.Current.Humidity, Equals, states[0].Humidity)
	c.Check(r.Current.Wind, Equals, states[0].Wind)
	c.Check(r.Current.VisibleLight, Equals, zeus.Light(-1000.0))
	c.Check(r.Current.UVLight, Equals, zeus.Light(-1000.0))
	c.Check(r.Next, IsNil)
	c.Check(r.NextTime, IsNil)

	c.Check(m.Close(), IsNil)

	_, ok = <-reports
	c.Check(ok, Equals, false)
	wg.Wait()

}
