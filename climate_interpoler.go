package main

import (
	"fmt"
	"math"
	"time"
)

type Interpolation interface {
	State(t time.Time) State
	String() string
}

type staticState State

func (s *staticState) State(time.Time) State {
	return State(*s)
}

func (s *staticState) String() string {
	return fmt.Sprintf("static state: %+v", State(*s))
}

type interpolation struct {
	start    time.Time
	from, to State
	duration time.Duration
}

func interpolate(from, to, completion float64) float64 {
	if from == math.Inf(-1) {
		if to == math.Inf(-1) {
			return math.Inf(-1)
		}
		return to
	} else if to == math.Inf(-1) {
		return from
	}
	return from + (to-from)*completion
}

func (i *interpolation) State(t time.Time) State {
	ellapsed := t.Sub(i.start)
	if ellapsed < 0 {
		ellapsed = 0
	} else if ellapsed > i.duration {
		ellapsed = i.duration
	}
	completion := float64(ellapsed.Seconds()) / float64(i.duration.Seconds())

	return State{
		Name:         fmt.Sprintf("%s to %s", i.from.Name, i.to.Name),
		Temperature:  Temperature(interpolate(i.from.Temperature.Value(), i.to.Temperature.Value(), completion)),
		Humidity:     Humidity(interpolate(i.from.Humidity.Value(), i.to.Humidity.Value(), completion)),
		Wind:         Wind(interpolate(i.from.Wind.Value(), i.to.Wind.Value(), completion)),
		VisibleLight: Light(interpolate(i.from.VisibleLight.Value(), i.to.VisibleLight.Value(), completion)),
		UVLight:      Light(interpolate(i.from.UVLight.Value(), i.to.UVLight.Value(), completion)),
	}
}

func (i *interpolation) String() string {
	return fmt.Sprintf("interpolation from '%s' to '%s' in %s", i.from.Name, i.to.Name, i.duration)
}

type ClimateInterpoler interface {
	StateAfter(t time.Time) (time.Time, Interpolation)
}
