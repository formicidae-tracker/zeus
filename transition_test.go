package main

import (
	"regexp"
	"time"

	. "gopkg.in/check.v1"
	yaml "gopkg.in/yaml.v2"
)

type TransitionSuite struct{}

var _ = Suite(&TransitionSuite{})

func (s *TransitionSuite) TestParsing(c *C) {
	testdata := []struct {
		Text       string
		Transition Transition
	}{
		{
			Text:       ``,
			Transition: Transition{},
		},
		{
			Text: `from: day
to: night
duration: 10m
start: 12:00
`,
			Transition: Transition{
				From:     "day",
				To:       "night",
				Duration: 10 * time.Minute,
				Start:    time.Date(0, 1, 1, 12, 00, 0, 0, time.UTC),
			},
		},
		{
			Text: `from: night
to: day
duration: 30m
start: 18:02
`,
			Transition: Transition{
				From:     "night",
				To:       "day",
				Duration: 30 * time.Minute,
				Start:    time.Date(0, 1, 1, 18, 02, 0, 0, time.UTC),
			},
		},
	}

	for _, d := range testdata {
		res := Transition{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, IsNil) == false {
			continue
		}
		c.Check(res, DeepEquals, d.Transition)
	}

	errordata := []struct {
		Text         string
		ErrorMatches string
	}{
		{
			Text: `start: 18:03:10
from: a
to: b
`,
			ErrorMatches: "parsing time \".*\": extra text: .*",
		},
		{
			Text:         `from: a`,
			ErrorMatches: "'from' and 'to' fields are required",
		},
		{
			Text:         `duration: aey`,
			ErrorMatches: "yaml: unmarshal errors:",
		},
	}

	for _, d := range errordata {
		res := Transition{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, Not(IsNil)) == true {
			rexp := regexp.MustCompile(d.ErrorMatches)
			c.Check(rexp.MatchString(err.Error()), Equals, true, Commentf("got error: `%s`\nregexp: `%s`", err, d.ErrorMatches))
		}
	}
}
