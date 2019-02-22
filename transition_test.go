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
				Day:      0,
			},
		},
		{
			Text: `from: night
to: day
duration: 30m
start: 18:02
day: 3
`,
			Transition: Transition{
				From:     "night",
				To:       "day",
				Duration: 30 * time.Minute,
				Start:    time.Date(0, 1, 1, 18, 02, 0, 0, time.UTC),
				Day:      3,
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
			Text: `from: a
start: 14:00`,
			ErrorMatches: "'From' and 'To' fields are required",
		},
		{
			Text:         `duration: aey`,
			ErrorMatches: "yaml: unmarshal errors:",
		},
		{
			Text: `start: 12:00
day: foo`,
			ErrorMatches: "strconv.Atoi: .*",
		},
		// 		{
		// 			Text: `from: a
		// to: b
		// start: 08:00
		// day: 3
		// start-time-delta: 3m`,
		// 			ErrorMatches: "StartTimeDelta is only available for recurring transitions",
		// 		},
	}

	for _, d := range errordata {
		res := Transition{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, Not(IsNil), Commentf("parsing '%s'", d.Text)) == true {
			rexp := regexp.MustCompile(d.ErrorMatches)
			c.Check(rexp.MatchString(err.Error()), Equals, true, Commentf("got error: `%s`\nregexp: `%s`", err, d.ErrorMatches))
		}
	}
}
