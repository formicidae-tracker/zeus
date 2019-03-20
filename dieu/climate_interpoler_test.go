package main

import (
	reflect "reflect"
	"time"

	"git.tuleu.science/fort/dieu"
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
				Temperature:  dieu.UndefinedTemperature,
				Humidity:     dieu.UndefinedHumidity,
				Wind:         dieu.UndefinedWind,
				VisibleLight: dieu.UndefinedLight,
				UVLight:      dieu.UndefinedLight,
			},
			"static state: {Name: Temperature:-Inf Humidity:-Inf Wind:-Inf VisibleLight:-Inf UVLight:-Inf}",
		},
		{
			&transition{
				from:     dieu.State{Name: "day"},
				to:       dieu.State{Name: "night"},
				duration: 30 * time.Minute,
				start:    time.Date(2019, 1, 1, 10, 00, 0, 0, time.UTC),
			},
			"transition from 'day' to 'night' in 30m0s at 2019-01-01 10:00:00 +0000 UTC",
		},
	}

	for _, d := range testdata {
		c.Check(d.i.String(), Equals, d.s)
	}
}

func (s *ClimateInterpolerSuite) TestStaticState(c *C) {
	testdata := []dieu.State{
		{
			Temperature:  dieu.UndefinedTemperature,
			Humidity:     30,
			Wind:         20,
			VisibleLight: dieu.UndefinedLight,
			UVLight:      45,
		},
	}

	for _, s := range testdata {
		c.Check((*staticState)(&s).State(time.Now()), Equals, s)
	}
}

func (s *ClimateInterpolerSuite) TestInterpolation(c *C) {

	i := transition{
		from: dieu.State{
			Name:         "a",
			Wind:         0,
			Temperature:  dieu.UndefinedTemperature,
			Humidity:     dieu.UndefinedHumidity,
			VisibleLight: 10,
		},
		to: dieu.State{
			Name:         "b",
			Wind:         20,
			Temperature:  dieu.UndefinedTemperature,
			Humidity:     40,
			VisibleLight: dieu.UndefinedLight,
		},
		duration: 30 * time.Minute,
	}

	testdata := []struct {
		d time.Duration
		s dieu.State
	}{
		{0, dieu.State{Wind: 0, Temperature: dieu.UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{15 * time.Minute, dieu.State{Wind: 10, Temperature: dieu.UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{30 * time.Minute, dieu.State{Wind: 20, Temperature: dieu.UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{40 * time.Minute, dieu.State{Wind: 20, Temperature: dieu.UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
		{-1 * time.Minute, dieu.State{Wind: 0, Temperature: dieu.UndefinedTemperature, Humidity: 40, VisibleLight: 10}},
	}

	for _, d := range testdata {
		t := time.Now()
		i.start = t
		d.s.Name = "a to b"
		c.Check(i.State(t.Add(d.d)), Equals, d.s)
	}

}

func (s *ClimateInterpolerSuite) TestClimateInterpoler(c *C) {

	definedDay := dieu.State{
		Name:         "day",
		Temperature:  26,
		Humidity:     dieu.UndefinedHumidity,
		Wind:         100,
		VisibleLight: 30,
		UVLight:      100,
	}
	definedNight := dieu.State{
		Name:         "night",
		Temperature:  22,
		Humidity:     60,
		Wind:         dieu.UndefinedWind,
		VisibleLight: 0,
		UVLight:      0,
	}
	definedDay2 := definedDay
	definedDay2.Name = "day2"
	definedDay2.Humidity = 70
	definedNight2 := definedNight
	definedNight2.Name = "night2"
	definedNight2.Humidity = dieu.UndefinedHumidity

	computedDay := definedDay
	computedDay.Humidity = definedNight.Humidity
	computedDay2 := computedDay
	computedDay2.Humidity = 70
	computedDay2.Name = "day2"
	computedNight := definedNight
	computedNight.Wind = definedDay.Wind
	computedNight2 := computedNight
	computedNight2.Name = "night2"
	computedNight2.Humidity = computedDay2.Humidity

	states := []dieu.State{definedDay, definedNight, definedDay2, definedNight2}

	transitions := []dieu.Transition{
		dieu.Transition{
			From:     "night",
			To:       "day",
			Start:    time.Date(0, 1, 1, 07, 30, 0, 0, time.UTC),
			Duration: 30 * time.Minute,
		},
		dieu.Transition{
			From:     "day",
			To:       "night",
			Start:    time.Date(0, 1, 1, 18, 30, 0, 0, time.UTC),
			Duration: 30 * time.Minute,
		},
		dieu.Transition{
			From:     "night2",
			To:       "day2",
			Start:    time.Date(0, 1, 1, 07, 40, 0, 0, time.UTC),
			Duration: 30 * time.Minute,
		},
		dieu.Transition{
			From:     "day2",
			To:       "night2",
			Start:    time.Date(0, 1, 1, 18, 20, 0, 0, time.UTC),
			Duration: 30 * time.Minute,
		},
		dieu.Transition{
			From:     "day",
			To:       "night2",
			Start:    time.Date(0, 1, 1, 18, 30, 0, 0, time.UTC),
			Duration: 20 * time.Minute,
			Day:      4,
		},
	}

	basedate := time.Date(2019, 1, 2, 0, 0, 0, 0, time.UTC)

	testdata := []struct {
		time                   time.Time
		nextInterpolationStart time.Time
		interpolation          Interpolation
		state                  dieu.State
	}{
		{
			basedate.Add(11 * time.Hour),
			basedate.Add(18*time.Hour + 30*time.Minute),
			(*staticState)(&computedDay),
			computedDay,
		},
		{
			basedate.Add(18*time.Hour + 45*time.Minute),
			basedate.Add(19 * time.Hour),
			&transition{
				from:     computedDay,
				to:       computedNight,
				start:    basedate.Add(18*time.Hour + 30*time.Minute),
				duration: 30 * time.Minute,
			},
			interpolateState(computedDay, computedNight, 0.5),
		},
		{
			basedate.Add(20 * time.Hour),
			basedate.AddDate(0, 0, 1).Add(7*time.Hour + 30*time.Minute),
			(*staticState)(&computedNight),
			computedNight,
		},
		{
			basedate.AddDate(0, 0, 1).Add(7*time.Hour + 40*time.Minute),
			basedate.AddDate(0, 0, 1).Add(8 * time.Hour),
			&transition{
				from:     computedNight,
				to:       computedDay,
				start:    basedate.AddDate(0, 0, 1).Add(7*time.Hour + 30*time.Minute),
				duration: 30 * time.Minute,
			},
			interpolateState(computedNight, computedDay, 1.0/3.0),
		},
		{
			basedate.AddDate(0, 0, 3).Add(18*time.Hour + 35*time.Minute),
			basedate.AddDate(0, 0, 3).Add(18*time.Hour + 50*time.Minute),
			&transition{
				from:     computedDay,
				to:       computedNight2,
				start:    basedate.AddDate(0, 0, 3).Add(18*time.Hour + 30*time.Minute),
				duration: 20 * time.Minute,
			},
			interpolateState(computedDay, computedNight2, 1.0/4.0),
		},
		{
			basedate.AddDate(0, 0, 4).Add(7*time.Hour + 50*time.Minute),
			basedate.AddDate(0, 0, 4).Add(8*time.Hour + 10*time.Minute),
			&transition{
				from:     computedNight2,
				to:       computedDay2,
				start:    basedate.AddDate(0, 0, 4).Add(7*time.Hour + 40*time.Minute),
				duration: 30 * time.Minute,
			},
			interpolateState(computedNight2, computedDay2, 1.0/3.0),
		},
		{
			basedate.AddDate(0, 0, 4).Add(18*time.Hour + 40*time.Minute),
			basedate.AddDate(0, 0, 4).Add(18*time.Hour + 50*time.Minute),
			&transition{
				from:     computedDay2,
				to:       computedNight2,
				start:    basedate.AddDate(0, 0, 4).Add(18*time.Hour + 20*time.Minute),
				duration: 30 * time.Minute,
			},
			interpolateState(computedDay2, computedNight2, 2.0/3.0),
		},
	}

	i, err := NewClimateInterpoler(states, transitions, basedate.Add(10*time.Hour))
	c.Assert(err, IsNil)

	for _, d := range testdata {
		interpolation, next, nextInterpolation := i.CurrentInterpolation(d.time)
		if c.Check(nextInterpolation, Not(IsNil), Commentf("Testing at %s", d.time)) == true {
			c.Check(next, Equals, d.nextInterpolationStart, Commentf("Testing at %s", d.time))
		}
		c.Check(interpolation, DeepEquals, d.interpolation, Commentf("Testing %s", d.time))
		c.Check(interpolation.State(d.time), DeepEquals, d.state, Commentf("Testing at %s", d.time))
	}

	i, err = NewClimateInterpoler(states, transitions, basedate.Add(-2*time.Hour))
	c.Assert(err, IsNil)
	interpolation, next, nextInterpolation := i.CurrentInterpolation(basedate.Add(-1 * time.Hour))
	c.Check(interpolation, DeepEquals, (*staticState)(&computedNight))
	if c.Check(nextInterpolation, Not(IsNil)) == true {
		c.Check(next, Equals, basedate.Add(7*time.Hour+30*time.Minute))
	}

	transitions[4].Start = time.Date(0, 1, 1, 18, 20, 0, 0, time.UTC)
	transitions = append(transitions, dieu.Transition{
		From:  "day",
		To:    "night",
		Start: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
	})
	transitions[5].Day = 4
	_, err = NewClimateInterpoler(states, transitions, basedate)
	c.Check(err, ErrorMatches, ".*Transition.* is shadowed by .*Transition.*")

	transitions[5].Start = time.Date(0, 1, 1, 18, 21, 0, 0, time.UTC)
	_, err = NewClimateInterpoler(states, transitions, basedate)
	c.Check(err, ErrorMatches, ".*Transition.* is shadowed by .*Transition.*")

	transitions = transitions[0:5]

	transitions[4].To = "night3"
	_, err = NewClimateInterpoler(states, transitions, basedate)
	c.Check(err, ErrorMatches, "Undefined state 'night3' in .*Transition.*")

	transitions[4].From = "day3"
	transitions[4].To = "night2"
	_, err = NewClimateInterpoler(states, transitions, basedate)
	c.Check(err, ErrorMatches, "Undefined state 'day3' in .*Transition.*")

	transitions[4].From = "day"
	states[3].Name = "day"
	_, err = NewClimateInterpoler(states, transitions, basedate)
	c.Check(err, ErrorMatches, "Cannot redefine state 'day'")

}

func (s *ClimateInterpolerSuite) TestBackAndForthWalk(c *C) {
	states := []dieu.State{
		dieu.State{Name: "a"},
		dieu.State{Name: "b"},
	}

	stdDuration := 30 * time.Minute
	transitions := []dieu.Transition{
		dieu.Transition{
			From:     "a",
			To:       "b",
			Start:    time.Date(0, 1, 1, 23, 45, 0, 0, time.UTC),
			Duration: stdDuration,
		},
		dieu.Transition{
			From:     "b",
			To:       "a",
			Start:    time.Date(0, 1, 1, 5, 45, 0, 0, time.UTC),
			Duration: stdDuration,
		},
		dieu.Transition{
			From:     "a",
			To:       "b",
			Start:    time.Date(0, 1, 1, 11, 45, 0, 0, time.UTC),
			Duration: stdDuration,
		},
		dieu.Transition{
			From:     "b",
			To:       "a",
			Start:    time.Date(0, 1, 1, 17, 45, 0, 0, time.UTC),
			Duration: stdDuration,
		},
	}

	dater := func(day, hour, minute int) time.Time {
		return time.Date(2019, 6, 15+day, hour, minute, 0, 0, time.UTC)
	}

	interpoler, err := NewClimateInterpoler(states, transitions, dater(0, 17, 0))
	c.Assert(err, IsNil)

	expected := []Interpolation{
		&staticState{Name: states[1].Name},
		&transition{start: dater(0, 17, 45), from: states[1], to: states[0], duration: stdDuration},
		&staticState{Name: states[0].Name},
		&transition{start: dater(0, 23, 45), from: states[0], to: states[1], duration: stdDuration},
		&staticState{Name: states[1].Name},
		&transition{start: dater(1, 5, 45), from: states[1], to: states[0], duration: stdDuration},
		&staticState{Name: states[0].Name},
		&transition{start: dater(1, 11, 45), from: states[0], to: states[1], duration: stdDuration},
		&staticState{Name: states[1].Name},
		&transition{start: dater(1, 17, 45), from: states[1], to: states[0], duration: stdDuration},
		&staticState{Name: states[0].Name},
		&transition{start: dater(1, 23, 45), from: states[0], to: states[1], duration: stdDuration},
		&staticState{Name: states[1].Name},
		&transition{start: dater(2, 5, 45), from: states[1], to: states[0], duration: stdDuration},
		&staticState{Name: states[0].Name},
		&transition{start: dater(2, 11, 45), from: states[0], to: states[1], duration: stdDuration},
	}

	expectedTimes := []time.Time{
		dater(0, 17, 45),
		dater(0, 18, 15),
		dater(0, 23, 45),
		dater(1, 00, 15),
		dater(1, 05, 45),
		dater(1, 06, 15),
		dater(1, 11, 45),
		dater(1, 12, 15),
		dater(1, 17, 45),
		dater(1, 18, 15),
		dater(1, 23, 45),
		dater(2, 00, 15),
		dater(2, 05, 45),
		dater(2, 06, 15),
		dater(2, 11, 45),
		dater(2, 12, 15),
	}
	c.Assert(len(expected), Equals, len(expectedTimes))

	maxDate := dater(2, 17, 00)
	currentDate := dater(0, 16, 1)
	currentInterpolation, _, _ := interpoler.CurrentInterpolation(currentDate)
	for i, e := range expected {

		for t := currentDate; t.Before(maxDate); t = t.Add(1 * time.Minute) {
			res, nextT, nextInterpolation := interpoler.CurrentInterpolation(t)
			c.Assert(nextInterpolation, Not(IsNil))

			if reflect.DeepEqual(currentInterpolation, res) == true {
				c.Check(res, DeepEquals, e)
				c.Check(nextT, Equals, expectedTimes[i], Commentf("%d %s %s", i, currentInterpolation, t))
			} else {
				currentDate = t
				currentInterpolation = res
				break
			}
		}
	}

}
