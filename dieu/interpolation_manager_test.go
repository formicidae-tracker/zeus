package main

import (
	"bytes"
	"sync"
	"time"

	"git.tuleu.science/fort/dieu"
	. "gopkg.in/check.v1"
)

type InterpolationManagerSuite struct {
}

var _ = Suite(&InterpolationManagerSuite{})

func (s *InterpolationManagerSuite) TestInterpolationReportsSanitized(c *C) {

	states := []dieu.State{
		dieu.State{
			Name:         "single and only one",
			Temperature:  22.0,
			Humidity:     50.0,
			Wind:         100.0,
			VisibleLight: dieu.UndefinedLight,
			UVLight:      dieu.UndefinedLight,
		},
	}

	reports := make(chan dieu.StateReport, 1)

	m, err := NewInterpolationManager("test-zone", states, []dieu.Transition{}, []capability{}, reports, bytes.NewBuffer(nil))
	c.Assert(err, IsNil)
	init := make(chan struct{})
	quit := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go m.Interpolate(&wg, init, quit)

	m.period = 1 * time.Millisecond

	close(init)
	r, ok := <-reports
	c.Assert(r.Current, Not(IsNil))
	c.Check(ok, Equals, true)
	c.Check(r.Current.Temperature, Equals, states[0].Temperature)
	c.Check(r.Current.Humidity, Equals, states[0].Humidity)
	c.Check(r.Current.Wind, Equals, states[0].Wind)
	c.Check(r.Current.VisibleLight, Equals, dieu.Light(-1000.0))
	c.Check(r.Current.UVLight, Equals, dieu.Light(-1000.0))
	c.Check(r.Next, IsNil)
	c.Check(r.NextTime, IsNil)

	time.Sleep(5 * m.period)

	close(quit)

	_, ok = <-reports
	c.Check(ok, Equals, false)

}
