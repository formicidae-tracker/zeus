package zeus

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	. "gopkg.in/check.v1"
)

type SeasonFileSuite struct {
}

var _ = Suite(&SeasonFileSuite{})

func (s *SeasonFileSuite) TestDeprecatedListing(c *C) {
	testdata := []struct {
		Input         string
		IsError       bool
		Name, Comment string
	}{
		{`emails:
  - noreply@unil.ch`, false, "emails", "value ignored"},
		{`zones:
  foo:
    can-interface: slcan0`, false, "zones.foo.can-interface", "value ignored"},
		{`zones:
  foo:
    devices-id: 1`, false, "zones.foo.devices-id", "value ignored"},
		{`zones:
  foo:
    climate-report-file: /foo/bar`, false, "zones.foo.climate-report-file", "climate logs are saved under `/data/fort-user/fort-experiments/climate/foo.<timestamp>.climate.txt`"},
	}

	for _, d := range testdata {
		lines, err := checkDeprecatedLines([]byte(d.Input))
		if c.Check(err, IsNil) == false || c.Check(len(lines), Equals, 1) == false {
			continue
		}
		c.Check(lines[0].isError, Equals, d.IsError)
		c.Check(lines[0].name, Equals, d.Name)
		c.Check(lines[0].comment, Equals, d.Comment)

	}
}

func (s *SeasonFileSuite) TestDeprecatedFormating(c *C) {
	lines := []deprecatedLine{
		{"foo", "comment", false},
		{"bar", "another comment", true},
	}
	buffer := bytes.NewBuffer(nil)
	err := formatDeprecatedLines(lines, buffer)
	c.Check(err.Error(), Equals, `invalid season file: bar is deprecated: another comment`)

	c.Check(len(buffer.Bytes()), Equals, 0)
	lines[1].isError = false
	c.Check(formatDeprecatedLines(lines, buffer), IsNil)
	c.Check(string(buffer.Bytes()), Matches, `time=".*" level=warning msg="deprecated field" comment=comment field=foo
time=".*" level=warning msg="deprecated field" comment="another comment" field=bar
`)
}

func (s *SeasonFileSuite) TestWritingShouldBeReadable(c *C) {
	season := SeasonFile{
		Zones: map[string]ZoneClimate{
			"box": {
				MinimalTemperature: 14.0,
				MaximalTemperature: 34.0,
				MinimalHumidity:    40.0,
				MaximalHumidity:    80.0,
				States: []State{
					{
						Name:         "day",
						Temperature:  16.0,
						Humidity:     60.0,
						Wind:         100,
						VisibleLight: 20,
						UVLight:      100,
					},
					{
						Name:         "night",
						Temperature:  16.0,
						Humidity:     UndefinedHumidity,
						Wind:         UndefinedWind,
						VisibleLight: 0,
						UVLight:      0,
					},
				},
				Transitions: []Transition{
					{
						From:     "day",
						To:       "night",
						Duration: 30 * time.Minute,
					},
					{
						From:     "night",
						To:       "day",
						Duration: 30 * time.Minute,
					},
				},
			},
		},
	}

	season.Zones["box"].Transitions[0].Start, _ = time.Parse("15:04", "17:00")
	season.Zones["box"].Transitions[1].Start, _ = time.Parse("15:04", "06:00")

	tmpdir, err := ioutil.TempDir("", "zeus-tests-season")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "test.season")
	c.Assert(season.WriteFile(filename), IsNil)
	defer func() {
		if c.Failed() == false {
			return
		}
		content, err := ioutil.ReadFile(filename)
		c.Assert(err, IsNil)
		log.Printf("written file:\n%s", content)
	}()

	result, err := ReadSeasonFile(filename, bytes.NewBuffer(nil))
	c.Check(err, IsNil)
	c.Check(*result, DeepEquals, season)
}
