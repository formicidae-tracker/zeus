package main

import (
	"bytes"
	"math"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
	. "gopkg.in/check.v1"
)

type InterpolationManagerSuite struct {
}

var _ = Suite(&InterpolationManagerSuite{})

func (s *InterpolationManagerSuite) TestInterpolationReportsUndefined(c *C) {

	states := []zeus.State{{
		Name:         "single and only one",
		Temperature:  22.0,
		Humidity:     50.0,
		Wind:         100.0,
		VisibleLight: zeus.UndefinedLight,
		UVLight:      zeus.UndefinedLight,
	}}

	i, err := NewInterpoler("test-zone", states, []zeus.Transition{})
	c.Assert(err, IsNil)

	i.(*interpoler).logger.SetOutput(bytes.NewBuffer(nil))
	i.(*interpoler).Period = 1 * time.Millisecond
	wg := sync.WaitGroup{}
	wg.Add(1)
	ready := make(chan struct{})
	go func() {
		i.Interpolate(ready)
		wg.Done()
	}()
	<-ready

	r, ok := <-i.Reports()
	c.Assert(r.Current, Not(IsNil))
	c.Check(ok, Equals, true)
	c.Check(r.Current.Temperature, Equals, states[0].Temperature)
	c.Check(r.Current.Humidity, Equals, states[0].Humidity)
	c.Check(r.Current.Wind, Equals, states[0].Wind)
	c.Check(r.Current.VisibleLight, Equals, zeus.Light(math.Inf(-1)))
	c.Check(r.Current.UVLight, Equals, zeus.Light(math.Inf(-1)))
	c.Check(r.Next, IsNil)
	c.Check(r.NextTime, IsNil)

	c.Check(i.Close(), IsNil)

	_, ok = <-i.Reports()
	c.Check(ok, Equals, false)
	wg.Wait()

}
