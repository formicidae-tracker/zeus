package main

import (
	"math"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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

	_, hook := test.NewNullLogger()

	i.(*interpoler).logger.Logger.AddHook(hook)
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

	c.Check(len(hook.Entries), Equals, 2)

	for _, e := range hook.Entries {
		c.Check(e.Level, Equals, logrus.InfoLevel)
		c.Check(e.Data["domain"], Equals, "zone/test-zone/climate")
	}

	c.Check(len(hook.Entries[0].Data), Equals, 2)
	c.Check(hook.Entries[0].Message, Equals, "starting interpolation loop")
	c.Check(hook.Entries[0].Data["interpolation"], Matches, "static state: {Name:single and only one .*}")

	c.Check(len(hook.Entries[1].Data), Equals, 1)
	c.Check(hook.Entries[1].Message, Equals, "stopping interpolation loop")
}
