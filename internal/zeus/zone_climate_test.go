package zeus

import (
	"time"

	. "gopkg.in/check.v1"
	yaml "gopkg.in/yaml.v2"
)

type ZoneClimateSuite struct{}

var _ = Suite(&ZoneClimateSuite{})

func (s *ZoneClimateSuite) TestParsing(c *C) {
	testdata := []struct {
		Text string
		Zone ZoneClimate
	}{
		{
			Text: `
minimal-temperature: 24.0
maximal-temperature: 31.0
minimal-humidity: 40.0
maximal-humidity: 80.0
states:
  - name: day
    temperature: 29.0
    humidity: 70.0
    wind: 100.0
    visible-light: 100.0
    uv-light: 100.0
  - name: night
    temperature: 26.0
    visible-light: 0.0
    uv-light: 0.0
transitions:
  - from: night
    to: day
    start: 06:00
    duration: 45m38s
  - from: day
    to: night
    start: 18:00
    duration: 1h03m1s
`,
			Zone: ZoneClimate{
				MinimalTemperature: 24,
				MaximalTemperature: 31,
				MinimalHumidity:    40,
				MaximalHumidity:    80,
				States: []State{
					State{
						Name:         "day",
						Temperature:  29.0,
						Humidity:     70.0,
						Wind:         100.0,
						VisibleLight: 100.0,
						UVLight:      100.0,
					},
					State{
						Name:         "night",
						Temperature:  26.0,
						Humidity:     UndefinedHumidity,
						Wind:         UndefinedWind,
						VisibleLight: 0.0,
						UVLight:      0.0,
					},
				},
				Transitions: []Transition{
					Transition{
						From:     "night",
						To:       "day",
						Start:    time.Date(0, 1, 1, 6, 00, 0, 0, time.UTC),
						Duration: 45*time.Minute + 38*time.Second,
					},
					Transition{
						From:     "day",
						To:       "night",
						Start:    time.Date(0, 1, 1, 18, 00, 0, 0, time.UTC),
						Duration: 1*time.Hour + 3*time.Minute + 1*time.Second,
					},
				},
			},
		},
	}

	for _, d := range testdata {
		res := ZoneClimate{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, IsNil) == false {
			continue
		}
		c.Check(res, DeepEquals, d.Zone)
	}
}
