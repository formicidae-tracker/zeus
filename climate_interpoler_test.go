package main

import (
	"time"

	. "gopkg.in/check.v1"
)

type ClimateInterpolerSuite struct {
}

var _ = Suite(&ClimateInterpolerSuite{})

func (s *ClimateInterpolerSuite) TestInterpolationFormat(c *C) {
	testdata := []struct {
		i Interpolation
		s string
	}{
		{
			&staticState{
				Temperature:  UndefinedTemperature,
				Humidity:     UndefinedHumidity,
				Wind:         UndefinedWind,
				VisibleLight: UndefinedLight,
				UVLight:      UndefinedLight,
			},
			"static state: {Name: Temperature:-Inf Humidity:-Inf Wind:-Inf VisibleLight:-Inf UVLight:-Inf}",
		},
		{
			&interpolation{
				from:     State{Name: "day"},
				to:       State{Name: "night"},
				duration: 30 * time.Minute,
			},
			"interpolation from 'day' to 'night' in 30m0s",
		},
	}

	for _, d := range testdata {
		c.Check(d.i.String(), Equals, d.s)
	}
}

func (s *ClimateInterpolerSuite) TestStaticState(c *C) {
	testdata := []State{
		{
			Temperature:  UndefinedTemperature,
			Humidity:     30,
			Wind:         20,
			VisibleLight: UndefinedLight,
			UVLight:      45,
		},
	}

	for _, s := range testdata {
		c.Check((*staticState)(&s).State(time.Now()), Equals, s)
	}
}

func (s *ClimateInterpolerSuite) TestInterpolation(c *C) {

	i := interpolation{
		from: State{
			Name:         "a",
			Wind:         0,
			Temperature:  UndefinedTemperature,
			Humidity:     UndefinedHumidity,
			VisibleLight: 10,
		},
		to: State{
			Name:         "b",
			Wind:         20,
			Temperature:  UndefinedTemperature,
			Humidity:     40,
			VisibleLight: UndefinedLight,
		},
		duration: 30 * time.Minute,
	}

	testdata := []struct {
		d time.Duration
		s State
	}{
		{0, State{Wind: 0, Temperature: UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{15 * time.Minute, State{Wind: 10, Temperature: UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{30 * time.Minute, State{Wind: 20, Temperature: UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{40 * time.Minute, State{Wind: 20, Temperature: UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{-1 * time.Minute, State{Wind: 0, Temperature: UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
	}

	for _, d := range testdata {
		t := time.Now()
		i.start = t
		d.s.Name = "a to b"
		c.Check(i.State(t.Add(d.d)), Equals, d.s)
	}

}
