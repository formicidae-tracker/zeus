package main

import (
	"regexp"
	"testing"

	yaml "gopkg.in/yaml.v2"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type StateSuite struct{}

var _ = Suite(&StateSuite{})

func checkEqualunit(actual, expected BoundedUnit, c *C) {
	if IsUndefined(expected) {
		c.Check(IsUndefined(actual), Equals, true)
	} else {
		c.Check(actual.Value(), Equals, expected.Value())
	}
}

func (s *StateSuite) TestParsing(c *C) {
	testdata := []struct {
		Text  string
		State State
	}{
		{
			Text: `temperature: 23.0`,
			State: State{
				Temperature:  23.0,
				Humidity:     UndefinedHumidity,
				Wind:         UndefinedWind,
				VisibleLight: UndefinedLight,
				UVLight:      UndefinedLight,
			},
		},
		{
			Text: `humidity: 45.0`,
			State: State{
				Temperature:  UndefinedTemperature,
				Humidity:     45.0,
				Wind:         UndefinedWind,
				VisibleLight: UndefinedLight,
				UVLight:      UndefinedLight,
			},
		},
	}

	for _, d := range testdata {
		res := State{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, IsNil) == false {
			continue
		}
		checkEqualunit(res.Temperature, d.State.Temperature, c)
		checkEqualunit(res.Humidity, d.State.Humidity, c)
		checkEqualunit(res.Wind, d.State.Wind, c)
		checkEqualunit(res.VisibleLight, d.State.VisibleLight, c)
		checkEqualunit(res.UVLight, d.State.UVLight, c)

	}

	res := State{}
	err := yaml.Unmarshal([]byte(`temperature: "very hot"`), &res)
	rx := regexp.MustCompile(`yaml:.*`)
	c.Check(rx.MatchString(err.Error()), Equals, true)

}
