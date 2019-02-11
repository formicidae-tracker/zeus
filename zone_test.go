package main

import (
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s *ZoneSuite) TestParsing(c *C) {
	testdata := []struct {
		Text string
		Zone Zone
	}{
		{
			Text: `devices:
  - class: "Zeus"
    can-interface: "slcan0"
    id: 1
  - class: "Celaeno"
    can-interface: "slcan0"
    id: 1
  - class: "Helios"
    can-interface: "slcan0"
    id: 1
climate-report-file: /data/someuser/my-experiment.txt
minimal-temperature: 24.0
maximal-temperature: 31.0
minimal-humidity: 40.0
maximal-humidity: 80.0
states:
  day:
    temperature: 29.0
    humidity: 70.0
    wind: 100.0
    visible-light: 100.0
    uv-light: 100.0
  night:
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
			Zone: Zone{
				Devices: []DeviceDefinition{
					DeviceDefinition{
						Class:        "Zeus",
						ID:           1,
						CANInterface: "slcan0",
					},
					DeviceDefinition{
						Class:        "Celaeno",
						ID:           1,
						CANInterface: "slcan0",
					},
					DeviceDefinition{
						Class:        "Helios",
						ID:           1,
						CANInterface: "slcan0",
					},
				},
				ClimateReportFile: "/data/someuser/my-experiment.txt",
			},
		},
	}

	for _, d := range testdata {
		res := Zone{}
		err := yaml.Unmarshal([]byte(d.Text), &res)
		if c.Check(err, IsNil) == false {
			continue
		}
		c.Check(res, DeepEquals, d.Zone)
	}
}
