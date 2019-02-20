package main

import (
	"math"
	"regexp"
	"testing"

	yaml "gopkg.in/yaml.v2"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type StateSuite struct{}

var _ = Suite(&StateSuite{})

func (s *StateSuite) TestParsing(c *C) {
	testdata := []struct {
		Text  string
		State State
	}{
		{
			Text: `temperature: 23.0`,
			State: State{
				Temperature:  23.0,
				Humidity:     Humidity(math.Inf(-1)),
				Wind:         Wind(math.Inf(-1)),
				VisibleLight: Light(math.Inf(-1)),
				UVLight:      Light(math.Inf(-1)),
			},
		},
		{
			Text: `humidity: 45.0`,
			State: State{
				Temperature:  Temperature(math.Inf(-1)),
				Humidity:     45.0,
				Wind:         Wind(math.Inf(-1)),
				VisibleLight: Light(math.Inf(-1)),
				UVLight:      Light(math.Inf(-1)),
			},
		},
	}

	for _, d := range testdata {
		res := State{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, IsNil) == false {
			continue
		}
		c.Check(res, DeepEquals, d.State)
	}

	res := State{}
	err := yaml.Unmarshal([]byte(`temperature: "very hot"`), &res)
	rx := regexp.MustCompile(`yaml:.*`)
	c.Check(rx.MatchString(err.Error()), Equals, true)

}
