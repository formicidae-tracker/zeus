package zeus

import (
	"bytes"

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
	c.Check(err.Error(), Equals, `invalid season file:
WARNING: 'foo' is deprecated (will raise an error in a future release): comment
ERROR: 'bar' is deprecated: another comment
`)
	c.Check(len(buffer.Bytes()), Equals, 0)
	lines[1].isError = false
	c.Check(formatDeprecatedLines(lines, buffer), IsNil)
	c.Check(string(buffer.Bytes()), Equals, `WARNING: 'foo' is deprecated (will raise an error in a future release): comment
WARNING: 'bar' is deprecated (will raise an error in a future release): another comment
`)
}
